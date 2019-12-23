package main

import (
	"context"
	pb "edgekv/frontend"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"
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

	mu sync.Mutex
}

// Get returns the Value at the specified key and data type
func (s *frontendServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	return &pb.GetResponse{Value: "dummy value", Size: 64}, nil
}

// Put stores the Value at the specified key and data type
func (s *frontendServer) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	return &pb.PutResponse{Status: 0}, nil
}

// Del deletes the key-value pair
func (s *frontendServer) Del(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	return &pb.DeleteResponse{Status: 0}, nil
}

func newServer() *frontendServer {
	s := &frontendServer{}
	return s
}

func main() {
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
	pb.RegisterFrontendServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}
