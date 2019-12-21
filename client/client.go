package main

import (
	"context"
	pb "edgekv/frontend"
	"flag"
	"log"
	"time"

	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("server_addr", "localhost:2379", "The server address in the format of host:port")
)

const (
	LOCALDATA  = true
	GLOBALDATA = false
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

func main() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewFrontendClient(conn)

	get(client, &pb.GetRequest{Key: "10", Type: LOCALDATA})
}
