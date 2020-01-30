package dht

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/ksonbol/edgekv/backend/backend"
	"github.com/ksonbol/edgekv/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client is the dht client
type Client struct {
	conn       *grpc.ClientConn
	rpcClient  pb.BackendClient
	rpcTimeout time.Duration
}

// NewClient creates a dht client with server address and credentials
func NewClient(serverAddr string, tls bool, caFile string,
	serverHostOverride string) (*Client, error) {
	var opts []grpc.DialOption
	if tls {
		// if caFile == "" {
		// 	set default caFile path
		// }
		creds, err := credentials.NewClientTLSFromFile(caFile, serverHostOverride)
		if err != nil {
			log.Fatalf("Failed to create TLS credentials %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	client := pb.NewBackendClient(conn)
	return &Client{conn: conn, rpcClient: client, rpcTimeout: 10 * time.Second}, err
}

// NewInsecureClient returns a dht client connected without credentials
// Do not use in production!
func NewInsecureClient(serverAddr string) (*Client, error) {
	return NewClient(serverAddr, false, "", "")
}

// Close the client gRPC connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// GetSuccessor returns the successor of the server
func (c *Client) GetSuccessor() (*Node, error) {
	// TODO: should we change this timeout?
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.EmptyReq{}
	res, err := c.rpcClient.GetSuccessor(ctx, req)
	return NewRemoteNode(res.GetAddr(), res.GetId()), err
}

// GetPredecessor returns the predecessor of the server
func (c *Client) GetPredecessor() (*Node, error) {
	// TODO: should we change this timeout?
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.EmptyReq{}
	res, err := c.rpcClient.GetPredecessor(ctx, req)
	return NewRemoteNode(res.GetAddr(), res.GetId()), err
}

// SetPredecessor sets the predecessor of the server
func (c *Client) SetPredecessor(node *Node) error {
	// TODO: should we change this timeout?
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.Node{Addr: node.Addr, Id: node.ID}
	_, err := c.rpcClient.SetPredecessor(ctx, req)
	return err
}

// FindSuccessor finds the successor of id
func (c *Client) FindSuccessor(id string) (*Node, error) {
	// TODO: should we change this timeout?
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.ID{Id: id}
	res, err := c.rpcClient.FindSuccessor(ctx, req)
	return NewRemoteNode(res.GetAddr(), res.GetId()), err
}

// GetKV gets the value associated with a key from the remote node
func (c *Client) GetKV(key string) (string, error) {
	// TODO: should we change this timeout?
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.GetRequest{Key: key}
	res, err := c.rpcClient.GetKV(ctx, req)
	return res.GetValue(), err
}

// PutKV adds the key-value pair or updates the value of given key
func (c *Client) PutKV(key string, value string) error {
	// TODO: should we change this timeout?
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.PutRequest{Key: key, Value: value}
	_, err := c.rpcClient.PutKV(ctx, req)
	return err
}

// DelKV removes the key-value pair from kv store
func (c *Client) DelKV(key string, dataType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.DeleteRequest{Key: key}
	res, err := c.rpcClient.DelKV(ctx, req)
	st := res.GetStatus()
	returnErr := err
	if err == nil {
		switch st {
		case utils.KeyNotFound:
			returnErr = fmt.Errorf("delete failed: could not find key: %s", key)
		case utils.UnknownError:
			returnErr = fmt.Errorf("delete operation failed: %v", err)
		}
	}
	return returnErr
}
