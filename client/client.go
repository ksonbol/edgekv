package main

import (
	"context"
	"edgekv/utils"
	"flag"
	"fmt"
	"log"
	"time"

	pb "edgekv/frontend/frontend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	serverAddr         = flag.String("server_addr", "localhost:2381", "The server address in the format of host:port")
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name used to verify the hostname returned by TLS handshake")
)

// Client is the user endpoint access to edgekv
type Client interface {
	// Close the client gRPC connection
	Close() error

	// Get the value associated with a key and data type from kv store
	Get(key string, dataType bool) (string, error)

	// Put adds the key-value pair or updates the value of given key
	Put(key string, dataType bool, value string) error

	// Del delete key from server's key-value store
	Del(key string, dataType bool) error
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

// InsecureEdgekvClient returns a client connected without credentials
// Do not use in production!
func NewInsecureEdgekvClient(serverAddr string) (*EdgekvClient, error) {
	return NewEdgekvClient(serverAddr, false, "", "")
}

// Close the client gRPC connection
func (c *EdgekvClient) Close() error {
	c.conn.Close()
	return nil
}

// Get the value associated with a key and data type from kv store
func (c *EdgekvClient) Get(key string, dataType bool) (string, error) {
	// TODO: should we change this timeout?
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.GetRequest{Key: key, Type: dataType}
	res, err := c.rpcClient.Get(ctx, req)
	return res.GetValue(), err
}

// Put adds the key-value pair or updates the value of given key
func (c *EdgekvClient) Put(key string, dataType bool, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.PutRequest{Key: key, Type: dataType, Value: value}
	_, err := c.rpcClient.Put(ctx, req)
	return err
}

// Del removes the key-value pair from kv store
func (c *EdgekvClient) Del(key string, dataType bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	req := &pb.DeleteRequest{Key: key, Type: dataType}
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

// run with flag -server_addr=localhost:PORT, default is localhost:2381
func main() {
	flag.Parse()
	cl, err := NewEdgekvClient(*serverAddr, *tls, *caFile, *serverHostOverride)
	if err != nil {
		fmt.Printf("Error while creating edgekv client %v\n", err)
	}
	key := "key"
	fmt.Println(cl.Get(key, utils.LocalData))
	fmt.Println(cl.Put(key, utils.LocalData, "val"))
	fmt.Println(cl.Get(key, utils.LocalData))
	fmt.Println(cl.Del(key, utils.LocalData))
	fmt.Println(cl.Get(key, utils.LocalData))
}
