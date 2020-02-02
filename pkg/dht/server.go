package dht

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	pb "github.com/ksonbol/edgekv/backend/backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Server is a DHT server
type Server struct {
	pb.UnimplementedBackendServer
	mux        sync.RWMutex
	hostname   string
	port       int
	node       *Node
	grpcServer *grpc.Server
}

// NewServer return a new DHT server
func NewServer(hostname string, port int, node *Node) *Server {
	s := &Server{
		hostname: hostname,
		port:     port,
		node:     node}
	return s
}

// Run the dht server
func (s *Server) Run(tls bool, certFile string, keyFile string) error {
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
	s.grpcServer = grpc.NewServer(opts...)
	pb.RegisterBackendServer(s.grpcServer, s)
	s.grpcServer.Serve(lis)
	return nil
}

// RunInsecure run the dht server without authentication
func (s *Server) RunInsecure() error {
	return s.Run(false, "", "")
}

// GetSuccessor returns successor of this node
func (s *Server) GetSuccessor(ctx context.Context, req *pb.EmptyReq) (*pb.Node, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	n := s.node.Successor()
	return &pb.Node{Id: n.ID, Addr: n.Addr}, nil
}

// GetPredecessor returns predecessor of this node
func (s *Server) GetPredecessor(ctx context.Context, req *pb.EmptyReq) (*pb.Node, error) {
	n := s.node.Predecessor()
	return &pb.Node{Id: n.ID, Addr: n.Addr}, nil
}

// SetPredecessor returns predecessor of this node
func (s *Server) SetPredecessor(ctx context.Context, req *pb.Node) (*pb.EmptyRes, error) {
	n := NewRemoteNode(req.GetAddr(), req.GetId())
	s.node.SetPredecessor(n)
	return &pb.EmptyRes{}, nil
}

// FindSuccessor returns predecessor of this node
func (s *Server) FindSuccessor(ctx context.Context, req *pb.ID) (*pb.Node, error) {
	n, err := s.node.findPredecessor(req.GetId())
	if err != nil {
		return nil, err
	}
	succ, err := n.GetSuccessorRPC()
	if err != nil {
		return nil, err
	}
	return &pb.Node{Id: succ.ID, Addr: succ.Addr}, nil
	// // if id is in (Predecessor.ID, n.ID], id is responsibility of n
	// if inIntervalHex(req.GetId(), incID(s.node.Predecessor().ID), incID(s.node.ID)) {
	// 	return &pb.Node{Id: s.node.ID, Addr: s.node.Addr}, nil
	// } else {
	// 	// send anither RPC call?
	// }
	// return &pb.Node{Id: id, Addr: addr}, err
}

// ClosestPrecedingFinger returns the closest node preceding id
func (s *Server) ClosestPrecedingFinger(ctx context.Context, req *pb.ID) (*pb.Node, error) {
	n := s.node.closestPrecedingFinger(req.GetId())
	return &pb.Node{Id: n.ID, Addr: n.Addr}, nil
}

func (s *Server) stop() {
	s.grpcServer.GracefulStop() // wait for pending RPCs to finish, then stop
}
