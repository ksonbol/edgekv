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
func getEndpoints(isLocal bool) []string {
	var endpoints []string
	var endpointsStr string
	var exists bool
	localEnvVar := "LOCAL_ENDPOINTS"
	globalEnvVar := "GLOBAL_ENDPOINTS"
	if isLocal {
		endpointsStr, exists = os.LookupEnv(localEnvVar)
	} else {
		endpointsStr, exists = os.LookupEnv(globalEnvVar)

	}
	if !exists {
		log.Fatalf("%s and %s environment variables not set correctly!", localEnvVar, globalEnvVar)
	}
	endpoints = strings.Split(endpointsStr, ",")
	return endpoints
}

// NewClient returns a new reusable etcd client
func NewClient(isLocal bool) *clientv3.Client {
	endpoints := getEndpoints(isLocal)
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
		log.Fatalf("Failed to connect to etcd cluster of endpoints %v %v", endpoints, err)
	}
	return cli
}
