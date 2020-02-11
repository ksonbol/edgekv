package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/ksonbol/edgekv/frontend/frontend"
	"github.com/ksonbol/edgekv/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client is the user endpoint access to edgekv
type Client interface {
	// Close the client gRPC connection
	Close() error

	// Get the value associated with a key and data type from kv store
	Get(key string, dataType string) (string, error)

	// Get the value associated with a key and data type from kv store
	RangeGet(startKey string, endKey string, dataType string) (string, error)

	// Put adds the key-value pair or updates the value of given key
	Put(key string, dataType string, value string) error

	// Del delete key from server's key-value store
	Del(key string, dataType string) error
}

// EdgekvClient is the client of edgekv key-value store
type EdgekvClient struct {
	conn       *grpc.ClientConn
	rpcClient  pb.FrontendClient
	rpcTimeout time.Duration
}

// NewEdgekvClient creates an edgekv client with server address
func NewEdgekvClient(serverAddr string, tls bool, caFile string,
	serverHostOverride string) (*EdgekvClient, error) {
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
	client := pb.NewFrontendClient(conn)
	return &EdgekvClient{conn: conn, rpcClient: client, rpcTimeout: 10 * time.Second}, err
}

// NewInsecureEdgekvClient returns a client connected without credentials
// Do not use in production!
func NewInsecureEdgekvClient(serverAddr string) (*EdgekvClient, error) {
	return NewEdgekvClient(serverAddr, false, "", "")
}

// Close the client gRPC connection
func (c *EdgekvClient) Close() error {
	return c.conn.Close()
}

// Get the value associated with a key and data type from kv store
func (c *EdgekvClient) Get(key string, dataType string) (string, error) {
	// TODO: should we change this timeout?
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.GetRequest{Key: key, Type: isLocal(dataType)}
	res, err := c.rpcClient.Get(ctx, req)
	return res.GetValue(), err
}

// Put adds the key-value pair or updates the value of given key
func (c *EdgekvClient) Put(key string, dataType string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.PutRequest{Key: key, Type: isLocal(dataType), Value: value}
	_, err := c.rpcClient.Put(ctx, req)
	return err
}

// Del removes the key-value pair from kv store
func (c *EdgekvClient) Del(key string, dataType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.DeleteRequest{Key: key, Type: isLocal(dataType)}
	res, err := c.rpcClient.Del(ctx, req)
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

// RangeGet returns the keys in the specified range from the storage
func (c *EdgekvClient) RangeGet(startKey string, endKey string, dataType string) (map[string]string, error) {
	res := make(map[string]string)
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.RangeRequest{Start: startKey, End: endKey, Type: isLocal(dataType)}
	stream, err := c.rpcClient.RangeGet(ctx, req)
	if err != nil {
		return nil, err
	}
	for {
		kv, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("RangeGet failed, %v", err)
		}
		res[kv.GetKey()] = kv.GetValue()
	}
	return res, nil
}

// RangeDel removes the keys in the specified range from the storage
func (c *EdgekvClient) RangeDel(startKey string, endKey string, dataType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.RangeRequest{Start: startKey, End: endKey, Type: isLocal(dataType)}
	_, err := c.rpcClient.RangeDel(ctx, req)
	return err
}

func isLocal(dataType string) bool {
	switch dataType {
	case utils.LocalDataStr:
		return utils.LocalData
	default:
		return utils.GlobalData
	}
}
