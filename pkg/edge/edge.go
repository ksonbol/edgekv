package edge

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "github.com/ksonbol/edgekv/frontend/frontend"

	"github.com/ksonbol/edgekv/internal/etcdclient"
	"github.com/ksonbol/edgekv/utils"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func checkError(err error) error {
	var returnErr error
	if err == nil {
		return err
	}
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
	mu         sync.Mutex
	etcdClient *clientv3.Client
	hostname   string
	port       int
}

// Get returns the Value at the specified key and data type
func (s *FrontendServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	var val string
	// var exists bool
	var returnErr error = nil
	var returnRes *pb.GetResponse

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// TODO: local vs global data
	res, err := s.etcdClient.Get(ctx, req.GetKey())
	cancel()
	returnErr = checkError(err)
	if returnErr == nil {
		// TODO: what if Kvs returns more than one kv-pair, is that possible?
		// TODO: how to choose which KV store to choose from? can we do that?
		if len(res.Kvs) > 0 {
			kv := res.Kvs[0]
			val = string(kv.Value)
			// log.Printf("Key: %s, Value: %s\n", kv.Key, kv.Value)
			returnRes = &pb.GetResponse{Value: val, Size: int32(len(val))}
		} else {
			returnErr = status.Errorf(codes.NotFound, "Key Not Found: %s", req.GetKey())
		}
	}
	return returnRes, returnErr
}

// Put stores the Value at the specified key and data type
func (s *FrontendServer) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	var returnErr error = nil
	var returnRes *pb.PutResponse

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// TODO: local vs global data
	_, err := s.etcdClient.Put(ctx, req.GetKey(), req.GetValue())
	cancel()
	returnErr = checkError(err)
	if returnErr == nil {
		returnRes = &pb.PutResponse{Status: utils.KVAddedOrUpdated}
	}
	return returnRes, returnErr
}

// Del deletes the key-value pair
func (s *FrontendServer) Del(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	var returnErr error = nil
	var returnRes *pb.DeleteResponse
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// TODO: local vs global data
	delRes, err := s.etcdClient.Delete(ctx, req.GetKey())
	cancel() // as given in etcd docs, wouldnt do harm anyway
	returnErr = checkError(err)
	if returnErr == nil {
		if delRes.Deleted < 1 {
			returnRes = &pb.DeleteResponse{Status: utils.KeyNotFound}
		} else {
			returnRes = &pb.DeleteResponse{Status: utils.KVDeleted}
		}
	} else {
		returnRes = &pb.DeleteResponse{Status: utils.UnknownError}
	}
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
	return s.etcdClient.Close()
}

// NewEdgeServer return a new edge server
func NewEdgeServer(hostname string, port int) *FrontendServer {
	s := &FrontendServer{
		etcdClient: etcdclient.NewEtcdClient(),
		hostname:   hostname,
		port:       port}
	return s
}
