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
	go s.grpcServer.Serve(lis)
	return nil
}

// RunInsecure run the dht server without authentication
func (s *Server) RunInsecure() error {
	return s.Run(false, "", "")
}

// GetSuccessor returns successor of this node
func (s *Server) GetSuccessor(ctx context.Context, req *pb.EmptyReq) (*pb.Node, error) {
	n := s.node.Successor()
	return &pb.Node{Id: n.ID, Addr: n.Addr}, nil
}

// GetPredecessor returns predecessor of this node
func (s *Server) GetPredecessor(ctx context.Context, req *pb.EmptyReq) (*pb.Node, error) {
	n := s.node.Predecessor()
	if n == nil {
		return &pb.Node{Id: s.node.ID, Addr: s.node.Addr}, nil // return self
	}
	return &pb.Node{Id: n.ID, Addr: n.Addr}, nil
}

// FindSuccessor returns predecessor of this node
func (s *Server) FindSuccessor(ctx context.Context, req *pb.ID) (*pb.Node, error) {
	// todo get peer info from context (we need requester node ID)
	// todo send requester node ID in context?
	// senderID := (ctx.Value("senderId")).string
	// senderAddr := ctx.Value("senderAddr")
	// if s.node.Successor().ID == s.node.ID {
	// 	n := NewRemoteNode(senderAddr, senderID, s.node.Transport)
	// 	s.node.SetSuccessor(n)
	// }
	// peer, ok := peer.FromContext(ctx)
	succ, err := s.node.findSuccessor(req.GetId())
	if err != nil {
		return nil, err
	}
	return &pb.Node{Id: succ.ID, Addr: succ.Addr}, nil
}

// ClosestPrecedingFinger returns the closest node preceding id
func (s *Server) ClosestPrecedingFinger(ctx context.Context, req *pb.ID) (*pb.Node, error) {
	n := s.node.closestPrecedingFinger(req.GetId())
	return &pb.Node{Id: n.ID, Addr: n.Addr}, nil
}

// Notify this node of a possible predecessor change
func (s *Server) Notify(ctx context.Context, req *pb.Node) (*pb.EmptyRes, error) {
	idChars := s.node.Conf.IDChars
	pred := s.node.Predecessor()
	// if pred == s.node {
	// 	defer close(s.node.nodeJoinCh) // other nodes have joined the system
	// }
	// if pred is nil or n` in (pred, n)
	if pred.ID != req.GetId() { // for readability and to avoid uneeded calculations
		if (pred == s.node) || inInterval(req.GetId(), incID(pred.ID, idChars), s.node.ID) {
			new := NewRemoteNode(req.GetAddr(), req.GetId(), s.node.Transport, nil)
			log.Printf("Replacing node %s old predecessor (%s) with %s\n",
				s.node.ID, pred.ID, new.ID)
			s.node.SetPredecessor(new)
			// delete unnecessary keys in range (old Pred, new Pred]
			if err := s.node.rangeDelKV(incID(pred.ID, s.node.Conf.IDChars),
				incID(new.ID, s.node.Conf.IDChars)); err != nil {
				return &pb.EmptyRes{}, fmt.Errorf("Updated successor but failed to delete keys: %v", err)
			}
		}
	}
	s.node.closeOnce.Do(func() {
		close(s.node.nodeJoinCh) // other nodes have joined the system
	})
	return &pb.EmptyRes{}, nil
}

// GetKV returns the value associated with a key from the edge group
func (s *Server) GetKV(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	val, err := s.node.GetKV(req.GetKey())
	return &pb.GetResponse{Value: val, Size: int32(len(val))}, err
}

// PutKV adds or updates key-value pair in the edge group
func (s *Server) PutKV(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	err := s.node.PutKV(req.GetKey(), req.GetValue())
	return &pb.PutResponse{}, err
}

// DelKV removes the key-value pair from the edge group
func (s *Server) DelKV(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := s.node.DelKV(req.GetKey())
	return &pb.DeleteResponse{}, err
}

// RangeGetKV streams all kv-pairs in the range [startID, endID)
func (s *Server) RangeGetKV(req *pb.RangeGetRequest, stream pb.Backend_RangeGetKVServer) error {
	kvs, err := s.node.RangeGetKV(req.GetStart(), req.GetEnd())
	if err != nil {
		return err
	}
	for k, v := range kvs {
		kv := &pb.KV{Key: k, Value: v}
		if err := stream.Send(kv); err != nil {
			return err
		}
	}
	return nil
}

// CanStore returns true if the node can store the key and false otherwise
func (s *Server) CanStore(ctx context.Context, req *pb.CanStoreRequest) (*pb.CanStoreResponse, error) {
	ans := s.node.CanStore(req.GetKey())
	return &pb.CanStoreResponse{Answer: ans}, nil
}

// IsLeaving returns true if node is leaving the dht ring
func (s *Server) IsLeaving(ctx context.Context, req *pb.EmptyReq) (*pb.IsLeavingRes, error) {
	leaving := s.node.isLeaving()
	return &pb.IsLeavingRes{Leaving: leaving}, nil
}

func (s *Server) stop() {
	s.grpcServer.GracefulStop() // wait for pending RPCs to finish, then stop
}
