package edge

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	pb "github.com/ksonbol/edgekv/frontend/frontend"
	"github.com/ksonbol/edgekv/internal/etcdclient"
	"github.com/ksonbol/edgekv/pkg/dht"

	"github.com/ksonbol/edgekv/utils"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func checkError(err error) error {
	if err == nil {
		return err
	}
	var returnErr error
	if err == context.Canceled {
		returnErr = status.Errorf(codes.Canceled,
			"Error sending request to etcd: ctx is canceled by another routine: %v", err)
	} else if err == context.DeadlineExceeded {
		returnErr = status.Errorf(codes.DeadlineExceeded,
			"Error sending request to etcd: ctx is attached with a deadline that has exceeded: %v", err)
	} else if err == rpctypes.ErrEmptyKey {
		returnErr = status.Errorf(codes.InvalidArgument,
			"Error sending request to etcd: client-side error: %v", err)
	} else if ev, ok := status.FromError(err); ok {
		code := ev.Code()
		if code == codes.DeadlineExceeded {
			// server-side context might have timed-out first (due to clock skew)
			// while original client-side context is not timed-out yet
			returnErr = status.Errorf(codes.DeadlineExceeded,
				"Error sending request to etcd: Server side has timed out: %v", err)
		}
	} else {
		returnErr = status.Errorf(codes.Unknown,
			"Error sending request to etcd: bad cluster endpoints, which are not etcd servers: %v", err)
	}
	return returnErr
}

// FrontendServer used to manage edge nodes
type FrontendServer struct {
	pb.UnimplementedFrontendServer
	mu       sync.Mutex
	localSt  *clientv3.Client
	globalSt *clientv3.Client
	gateway  *dht.Node
	hostname string
	port     int
}

// NewEdgeServer return a new edge server
func NewEdgeServer(hostname string, port int) *FrontendServer {
	s := &FrontendServer{
		localSt:  etcdclient.NewClient(true),
		globalSt: etcdclient.NewClient(false),
		hostname: hostname,
		port:     port,
	}
	return s
}

// SetGateway sets the connection to the gateway node
func (s *FrontendServer) SetGateway(addr string) {
	s.gateway = dht.NewRemoteNode(addr, "", nil, nil)
}

// Get returns the Value at the specified key and data type
func (s *FrontendServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	var err error
	var returnErr error = nil
	var returnRes *pb.GetResponse
	var res *clientv3.GetResponse
	var val string
	sender, ok := peer.FromContext(ctx)
	if !ok {
		log.Fatal("Couldn't get requester info to the edge node")
	}
	senderAddr := sender.Addr.String()
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	key := req.GetKey()
	switch req.GetType() {
	case utils.LocalData:
		res, err = s.localSt.Get(ctx, key)
	case utils.GlobalData:
		if s.gateway == nil {
			log.Fatal("Get request failed: gateway node not initialized at the edge")
		}
		// if request is coming from the gateway node, we don't ask it again if key
		// is our responsiblity
		if senderAddr == s.gateway.Addr {
			res, err = s.globalSt.Get(ctx, key)
		} else {
			ans, er := s.gateway.CanStoreRPC(key)
			if er != nil {
				log.Fatalf("Get request failed: communication with gateway node failed")
			}
			if ans {
				res, err = s.globalSt.Get(ctx, key)
			} else {
				val, err = s.gateway.GetKVRPC(key)
			}
		}
	}
	// cancel()
	returnErr = checkError(err)
	if (res != nil) && (returnErr == nil) {
		// TODO: what if Kvs returns more than one kv-pair, is that possible?
		if len(res.Kvs) > 0 {
			kv := res.Kvs[0]
			val = string(kv.Value)
			// log.Printf("Key: %s, Value: %s\n", kv.Key, kv.Value)
			returnRes = &pb.GetResponse{Value: val, Size: int32(len(val))}
		} else {
			returnErr = status.Errorf(codes.NotFound, "Key Not Found: %s", req.GetKey())
		}
	} else {
		if returnErr == nil {
			// we already have the value from a remote group
			returnRes = &pb.GetResponse{Value: val, Size: int32(len(val))}
		}
	}
	return returnRes, returnErr
}

// Put stores the Value at the specified key and data type
func (s *FrontendServer) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	var returnErr error = nil
	var err error
	var returnRes *pb.PutResponse
	sender, ok := peer.FromContext(ctx)
	if !ok {
		log.Fatal("Couldn't get requester info to the edge node")
	}
	senderAddr := sender.Addr.String()
	key := req.GetKey()
	// var res *clientv3.PutResponse
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	switch req.GetType() {
	case utils.LocalData:
		_, err = s.localSt.Put(ctx, key, req.GetValue())
	case utils.GlobalData:
		if s.gateway == nil {
			log.Fatal("Put request failed: gateway node not initialized at the edge")
		}
		if senderAddr == s.gateway.Addr {
			_, err = s.globalSt.Put(ctx, key, req.GetValue())
		} else {
			ans, er := s.gateway.CanStoreRPC(key)
			if er != nil {
				log.Fatalf("Put request failed: communication with gateway node failed")
			}
			if ans {
				_, err = s.globalSt.Put(ctx, key, req.GetValue())
			} else {
				err = s.gateway.PutKVRPC(key, req.GetValue())
			}
		}
	}
	// cancel()
	returnErr = checkError(err)
	if returnErr == nil {
		returnRes = &pb.PutResponse{Status: utils.KVAddedOrUpdated}
	}
	return returnRes, returnErr
}

// Del deletes the key-value pair
func (s *FrontendServer) Del(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	var returnErr error = nil
	var err error
	var returnRes *pb.DeleteResponse
	// var res *clientv3.DeleteResponse
	sender, ok := peer.FromContext(ctx)
	if !ok {
		log.Fatal("Couldn't get requester info to the edge node")
	}
	senderAddr := sender.Addr.String()
	key := req.GetKey()
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	switch req.GetType() {
	case utils.LocalData:
		_, err = s.localSt.Delete(ctx, key)
	case utils.GlobalData:
		if senderAddr == s.gateway.Addr {
			_, err = s.globalSt.Delete(ctx, key)
		} else {
			ans, er := s.gateway.CanStoreRPC(key)
			if er != nil {
				log.Fatalf("Delete request failed: communication with gateway node failed")
			}
			if ans {
				_, err = s.globalSt.Delete(ctx, key)
			} else {
				err = s.gateway.DelKVRPC(key)
			}
		}
	}
	// cancel() // as given in etcd docs, wouldnt do harm anyway
	returnErr = checkError(err)
	if returnErr == nil {
		returnRes = &pb.DeleteResponse{Status: utils.KVDeleted}
	}
	// TODO: we can check these keys to decide if the key was not found or actually deleted
	// but not really important for now
	// Todo: find out this: can the key be not deleted and returnErr still be nil?
	// 	if res.Deleted < 1 {
	// 		returnRes = &pb.DeleteResponse{Status: utils.KeyNotFound}
	// 	} else {
	// 		returnRes = &pb.DeleteResponse{Status: utils.KVDeleted}
	// 	}
	// } else {
	// 	returnRes = &pb.DeleteResponse{Status: utils.UnknownError}
	// }
	return returnRes, returnErr
}

// Run the edge server and connect to the etcd cluster
func (s *FrontendServer) Run(tls bool, certFile string,
	keyFile string) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.hostname, s.port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if tls {
		// if caFile == "" {
		// 	set default caFile path
		// }
		// if keyFile == "" {
		// 	set default keyFile path
		// }
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterFrontendServer(grpcServer, s)
	grpcServer.Serve(lis)
	return nil
}

// RunInsecure run the edge server without authentication
func (s *FrontendServer) RunInsecure() error {
	return s.Run(false, "", "")
}

// Close the edge server and free its resources
func (s *FrontendServer) Close() error {
	err1 := s.localSt.Close()
	err2 := s.globalSt.Close()
	if (err1 != nil) || (err2 != nil) {
		return fmt.Errorf("failed to close etcd client %v, %v", err1, err2)
	}
	return nil
}
