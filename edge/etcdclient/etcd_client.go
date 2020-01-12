package etcdclient

import (
	"log"
	"os"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc"
	// grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
)

// getEndpoints return a slice of endpoint info as hostname:port
func getEndpoints() []string {
	var endpoints []string
	endpointsStr, exists := os.LookupEnv("ENDPOINTS")
	if !exists {
		log.Fatalf("ENDPOINTS environment variable not set!")
	}
	endpoints = strings.Split(endpointsStr, ",")
	return endpoints
}

// NewEtcdClient returns a new reusable etcd client
func NewEtcdClient() *clientv3.Client {
	endpoints := getEndpoints()
	cli, err := clientv3.New(clientv3.Config{
		// Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"},
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
		// TODO: do we need these two lines?
		// DialKeepAliveTime:    time.Second,
		// DialKeepAliveTimeout: 500 * time.Millisecond,

		// DialOptions: []grpc.DialOption{
		// grpc.WithUnaryInterceptor(grpcprom.UnaryClientInterceptor),
		// grpc.WithStreamInterceptor(grpcprom.StreamClientInterceptor),
		// },
		// MaxCallSendMsgSize =
		// MaxCallRecvMsgSize =
	})
	if err != nil {
		// handle error!
		log.Fatal(err)
	}
	return cli
}
