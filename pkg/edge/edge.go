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
	"github.com/ksonbol/edgekv/pkg/dht"

	"github.com/ksonbol/edgekv/utils"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key := req.GetKey()
	switch req.GetType() {
	case utils.LocalData:
		res, err = s.localSt.Get(ctx, key) // get the key itself, no hashes used
		// log.Println("All local keys in this edge group")
		// log.Println(s.localSt.Get(ctx, "", clientv3.WithPrefix()))
	case utils.GlobalData:
		isHashed := req.GetIsHashed()
		if s.gateway == nil {
			return returnRes, fmt.Errorf("RangeGet request failed: gateway node not initialized at the edge")
		}
		state, err := s.gateway.GetStateRPC()
		if err != nil {
			return returnRes, fmt.Errorf("failed to get state of gateway node")
		}
		if state < dht.Ready { // could happen if gateway node was created but didnt join dht
			return returnRes, fmt.Errorf("edge node %s not connected to dht yet or gateway node not ready", s.print())
		}
		// log.Println("All global keys in this edge group")
		// log.Println(s.globalSt.Get(ctx, "", clientv3.WithPrefix()))
		if !isHashed {
			key = s.gateway.Conf.IDFunc(key)
		}
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
	key := req.GetKey()
	// var res *clientv3.PutResponse
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	switch req.GetType() {
	case utils.LocalData:
		_, err = s.localSt.Put(ctx, key, req.GetValue())
	case utils.GlobalData:
		isHashed := req.GetIsHashed()
		if s.gateway == nil {
			return returnRes, fmt.Errorf("RangeGet request failed: gateway node not initialized at the edge")
		}
		state, err := s.gateway.GetStateRPC()
		if err != nil {
			return returnRes, fmt.Errorf("failed to get state of gateway node")
		}
		if state < dht.Joined { // in order to reach ready state, a node may need to put some keys initially
			return returnRes, fmt.Errorf("edge node %s not connected to dht yet", s.print())
		}
		if !isHashed {
			key = s.gateway.Conf.IDFunc(key)
		}
		ans, err := s.gateway.CanStoreRPC(key)
		if err != nil {
			log.Fatalf("communication with gateway node failed: %v\n", err)
		}
		if ans {
			// log.Printf("Node %s:%d Writing key %s to local edge group", s.hostname, s.port, key)
			_, err = s.globalSt.Put(ctx, key, req.GetValue())
		} else {
			// log.Printf("Node %s:%d Writing key %s to remote edge group", s.hostname, s.port, key)
			err = s.gateway.PutKVRPC(key, req.GetValue())
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
	key := req.GetKey()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	switch req.GetType() {
	case utils.LocalData:
		_, err = s.localSt.Delete(ctx, key)
	case utils.GlobalData:
		isHashed := req.GetIsHashed()
		if s.gateway == nil {
			return returnRes, fmt.Errorf("Del request failed: gateway node not initialized at the edge")
		}
		state, err := s.gateway.GetStateRPC()
		if err != nil {
			return returnRes, fmt.Errorf("failed to get state of gateway node")
		}
		if state < dht.Ready { // could happen if gateway node was created but didnt join dht
			return returnRes, fmt.Errorf("edge node %s not connected to dht yet or gateway node not ready", s.print())
		}
		if !isHashed {
			key = s.gateway.Conf.IDFunc(key)
		}
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

// RangeGet returns the KV-pairs in the specified range [Start, end) and specified storage
// For Global storage, given key range is searched in the current group only
// and the given keys should be the hashed keys
// For local storage, no hashing is used anyway
func (s *FrontendServer) RangeGet(req *pb.RangeRequest, stream pb.Frontend_RangeGetServer) error {
	var err error
	var returnErr error = nil
	var res, res2, res3 *clientv3.GetResponse
	var kvRes *pb.KV
	var more bool
	startKey := req.GetStart()
	endKey := req.GetEnd()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	switch req.GetType() {
	case utils.LocalData:
		if startKey > endKey {
			return fmt.Errorf("Range Get over local data failed: start > end")
		}
		res, err = s.localSt.Get(ctx, startKey, clientv3.WithRange(endKey)) // get the key itself, no hashes used
	case utils.GlobalData:
		// keys are assumed to be already hashed
		if s.gateway == nil {
			return fmt.Errorf("RangeGet request failed: gateway node not initialized at the edge")
		}
		state, err := s.gateway.GetStateRPC()
		if err != nil {
			return fmt.Errorf("failed to get state of gateway node")
		}
		if state < dht.Ready { // could happen if gateway node was created but didnt join dht
			return fmt.Errorf("edge node %s not connected to dht yet or gateway node not ready", s.print())
		}
		if startKey > endKey {
			// get results as: [startKey, MaxID) + MaxID + [0, endKey)
			// because etcd does not have knowledge of the ring and start must be < end for each range request
			more = true
			end1 := s.gateway.MaxID()
			start2 := s.gateway.ZeroID()
			res, err = s.globalSt.Get(ctx, startKey, clientv3.WithRange(end1))
			if returnErr = checkError(err); returnErr != nil {
				return status.Errorf(codes.Unknown, "Range Get failed with error: %v", returnErr)
			}
			// log.Printf("First res length %d", res.Count)
			res2, err = s.globalSt.Get(ctx, end1)
			if returnErr = checkError(err); returnErr != nil {
				return status.Errorf(codes.Unknown, "Range Get failed with error: %v", returnErr)
			}
			// log.Printf("Second res length %d", res2.Count)

			res3, err = s.globalSt.Get(ctx, start2, clientv3.WithRange(endKey))
			// log.Printf("Third res length %d", res3.Count)
		} else {
			res, err = s.globalSt.Get(ctx, startKey, clientv3.WithRange(endKey))
		}
	}
	returnErr = checkError(err)
	if returnErr != nil {
		return status.Errorf(codes.Unknown, "Range Get failed with error: %v", returnErr)
	}
	// if (res == nil) || (res.Count == 0) { // should be same as if len(res.Kvs) == 0
	// 	return nil
	// }
	for _, kv := range res.Kvs {
		kvRes = &pb.KV{Key: string(kv.Key), Value: string(kv.Value)}
		stream.Send(kvRes)
	}
	if more {
		if res2 != nil {
			for _, kv := range res2.Kvs {
				kvRes = &pb.KV{Key: string(kv.Key), Value: string(kv.Value)}
				stream.Send(kvRes)
			}
		}
		if res3 != nil {
			for _, kv := range res3.Kvs {
				kvRes = &pb.KV{Key: string(kv.Key), Value: string(kv.Value)}
				stream.Send(kvRes)
			}
		}
	}
	return nil
}

// RangeDel returns the KV-pairs in the specified range [Start, end) and specified storage
// For Global storage, given key range is searched in the current group only
// and the given keys should be the hashed keys
// For local storage, no hashing is used anyway
func (s *FrontendServer) RangeDel(ctx context.Context, req *pb.RangeRequest) (*pb.EmptyRes, error) {
	var err error
	var returnErr error = nil
	var returnRes = &pb.EmptyRes{}
	startKey := req.GetStart()
	endKey := req.GetEnd()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	switch req.GetType() {
	case utils.LocalData:
		if startKey > endKey {
			return returnRes, fmt.Errorf("Range Del over local data failed: start > end")
		}
		_, err = s.localSt.Delete(ctx, startKey, clientv3.WithRange(endKey)) // use the key itself, no hashes used
		return returnRes, err
	case utils.GlobalData:
		// keys are assumed to be already hashed
		if s.gateway == nil {
			return returnRes, fmt.Errorf("RangeGet request failed: gateway node not initialized at the edge")
		}
		state, err := s.gateway.GetStateRPC()
		if err != nil {
			return returnRes, fmt.Errorf("failed to get state of gateway node")
		}
		if state < dht.Ready { // could happen if gateway node was created but didnt join dht
			return returnRes, fmt.Errorf("edge node %s not connected to dht yet or gateway node not ready", s.print())
		}
		if startKey > endKey {
			// delete keys in ranges: [startKey, MaxID) + MaxID + [0, endKey)
			// because etcd does not have knowledge of the ring and start must be < end for each range request
			end1 := s.gateway.MaxID()
			start2 := s.gateway.ZeroID()
			_, err = s.globalSt.Delete(ctx, startKey, clientv3.WithRange(end1))
			if returnErr = checkError(err); returnErr != nil {
				return returnRes, status.Errorf(codes.Unknown, "Range Del failed with error: %v", returnErr)
			}
			_, err = s.globalSt.Delete(ctx, end1)
			if returnErr = checkError(err); returnErr != nil {
				return returnRes, status.Errorf(codes.Unknown, "Range Del failed with error: %v", returnErr)
			}
			_, err = s.globalSt.Delete(ctx, start2, clientv3.WithRange(endKey))
			if returnErr = checkError(err); returnErr != nil {
				return returnRes, status.Errorf(codes.Unknown, "Range Del failed with error: %v", returnErr)
			}
		} else {
			_, err = s.globalSt.Delete(ctx, startKey, clientv3.WithRange(endKey))
			if returnErr = checkError(err); returnErr != nil {
				return returnRes, status.Errorf(codes.Unknown, "Range Del failed with error: %v", returnErr)
			}
		}
	}
	return returnRes, nil
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

func (s *FrontendServer) print() string {
	return fmt.Sprintf("%s:%d", s.hostname, s.port)
}
