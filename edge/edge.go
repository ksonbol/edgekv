package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/testdata"

	"edgekv/edge/etcdclient"
	pb "edgekv/frontend/frontend"
	"edgekv/utils"
)

var (
	tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile = flag.String("cert_file", "", "The TLS cert file")
	keyFile  = flag.String("key_file", "", "The TLS key file")
	hostname = flag.String("hostname", "localhost", "The server hostname or public IP address")
	port     = flag.Int("port", 10000, "The server port")
)

//
type frontendServer struct {
	pb.UnimplementedFrontendServer
	kvLocal    map[string]string
	kvGlobal   map[string]string
	mu         sync.Mutex
	etcdClient *clientv3.Client
}

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

// Get returns the Value at the specified key and data type
func (s *frontendServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	var val string
	// var exists bool
	var returnErr error = nil
	var returnRes *pb.GetResponse

	// TODO: implement two KV Stores
	// switch req.GetType() {
	// case utils.LocalData:
	// 	val, exists = s.kvLocal[req.GetKey()]
	// case utils.GlobalData:
	// 	val, exists = s.kvGlobal[req.GetKey()]
	// }
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
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
func (s *frontendServer) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	var returnErr error = nil
	var returnRes *pb.PutResponse
	// switch req.GetType() {
	// case utils.LocalData:
	// 	s.kvLocal[req.GetKey()] = req.GetValue()
	// case utils.GlobalData:
	// 	s.kvGlobal[req.GetKey()] = req.GetValue()
	// }
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := s.etcdClient.Put(ctx, req.GetKey(), req.GetValue())
	cancel()
	returnErr = checkError(err)
	if returnErr == nil {
		returnRes = &pb.PutResponse{Status: utils.KVAddedOrUpdated}
	}
	return returnRes, returnErr
}

// Del deletes the key-value pair
func (s *frontendServer) Del(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	var returnErr error = nil
	var returnRes *pb.DeleteResponse
	// switch req.GetType() {
	// case utils.LocalData:
	// 	delete(s.kvLocal, req.GetKey())
	// case utils.GlobalData:
	// 	delete(s.kvGlobal, req.GetKey())
	// }
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := s.etcdClient.Delete(ctx, req.GetKey())
	cancel()
	returnErr = checkError(err)
	if returnErr == nil {
		returnRes = &pb.DeleteResponse{Status: utils.KVDeleted}
	}
	return returnRes, returnErr
}

func newServer() *frontendServer {
	s := &frontendServer{kvLocal: make(map[string]string), etcdClient: etcdclient.NewEtcdClient()}
	return s
}

func runEdgeServer(frServer *frontendServer) {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *hostname, *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = testdata.Path("server1.pem")
		}
		if *keyFile == "" {
			*keyFile = testdata.Path("server1.key")
		}
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterFrontendServer(grpcServer, frServer)
	grpcServer.Serve(lis)

}
func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM) // CTRL-C->SIGINT, kill $PID->SIGTERM
	server := newServer()
	go runEdgeServer(server)
	log.Println("Listening to client requests")
	<-sigs
	(*server).etcdClient.Close()
	log.Println("Stopping the server...")
	os.Exit(0)
}
