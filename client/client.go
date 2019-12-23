package main

import (
	"context"
	pb "edgekv/frontend"
	"flag"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("server_addr", "localhost:2379", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
)

const (
	localdata  = true
	globaldata = false
)

func get(client pb.FrontendClient, req *pb.GetRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.Get(ctx, req)
	if err != nil {
		log.Fatalf("%v.Get(_) = _, %v: ", client, err)
	}
	log.Printf("Value: %s, Size: %d\n", res.GetValue(), res.GetSize())
}

func put(client pb.FrontendClient, req *pb.PutRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.Put(ctx, req)
	if err != nil {
		log.Fatalf("%v.Put(_) = _, %v: ", client, err)
	}
	st := res.GetStatus()
	switch st {
	case 1:
		log.Println("Put operation completed successfully.")
	case -1:
		log.Printf("Put operation failed, %v\n", err)
	}
}

func del(client pb.FrontendClient, req *pb.DeleteRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.Del(ctx, req)
	if err != nil {
		log.Fatalf("%v.Del(_) = _, %v: ", client, err)
	}
	st := res.GetStatus()
	switch st {
	case 1:
		log.Println("Delete operation completed successfully.")
	case 2:
		log.Println("Delete failed: could not find key.")
	case -1:
		log.Printf("Put operation failed, %v\n", err)
	}
}

// run with flag -serveraddress=localhost:PORT, default is localhost:2379
func main() {
	flag.Parse()
	var opts []grpc.DialOption
	if *tls {
		if *caFile == "" {
			*caFile = testdata.Path("ca.pem")
		}
		creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
		if err != nil {
			log.Fatalf("Failed to create TLS credentials %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewFrontendClient(conn)

	get(client, &pb.GetRequest{Key: "10", Type: localdata})
}
