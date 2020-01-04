package main

import (
	"context"
	"edgekv/utils"
	"flag"
	"fmt"
	"log"
	"time"

	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"

	pb "edgekv/frontend/frontend"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("server_addr", "localhost:10000", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
)

// Get the value associated with a key and data type from kv store
func Get(client pb.FrontendClient, req *pb.GetRequest) (*pb.GetResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.Get(ctx, req)
	return res, err
}

// Put Add a new key-value pair
func Put(client pb.FrontendClient, req *pb.PutRequest) (*pb.PutResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.Put(ctx, req)
	return res, err
	// if err != nil {
	// log.Fatalf("%v.Put(_) = _, %v: ", client, err)
	// }
	// return res, nil
}

// Del delete key from server's key-value store
func Del(client pb.FrontendClient, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.Del(ctx, req)
	if err != nil {
		log.Fatalf("%v.Del(_) = _, %v: ", client, err)
	}
	return res, nil
}

// Functions to use in the benchmark

// get just send the value or error to stdout
// TODO: return value instead and use it in benchmark
func get(client pb.FrontendClient, dataT bool, key string) (string, error) {
	var val string
	var returnErr error
	getRes, err := Get(client, &pb.GetRequest{Key: key, Type: dataT})
	// return getRes.GetValue(), err
	if err != nil {
		returnErr = fmt.Errorf("get operation failed: %v", err)
		// if st, ok := status.FromError(err); ok {
		// 	code := st.Code()
		// 	if code == codes.NotFound {
		// 		returnErr = fmt.Errorf("get operation failed: %v", err)
		// 	}
		// } else {
		// 	returnErr = fmt.Errorf(err)
		// }
	} else {
		// log.Printf("Value: %s, Size: %d\n", getRes.GetValue(), getRes.GetSize())
		val = getRes.GetValue()
	}
	// fmt.Printf("Value: %s, Size: %d\n", getRes.GetValue(), getRes.GetSize())
	return val, returnErr
}

// put just prints the output or error to stdout
// TODO: return value instead and use it in benchmark
func put(client pb.FrontendClient, dataT bool, key string, val string) (int32, error) {
	var returnSt int32 // return status
	var returnErr error
	putRes, err := Put(client, &pb.PutRequest{Key: key, Type: dataT, Value: val})
	returnSt = putRes.GetStatus()
	returnErr = err
	if err != nil {
		returnErr = fmt.Errorf("Put operation failed: %v", err)
	} else {
		switch returnSt {
		case utils.KVAddedOrUpdated:
			// log.Println("Put operation completed successfully.")
		case utils.UnknownError:
			// log.Printf("Put operation failed: %v.\n", err)
			returnErr = fmt.Errorf("Unkown error happened, put operation failed")

		}
	}
	return returnSt, returnErr
}

// put just prints the output or error to stdout
// TODO: return value instead and use it in benchmark
func del(client pb.FrontendClient, dataT bool, key string) (int32, error) {
	var returnSt int32
	var returnErr error
	delRes, err := Del(client, &pb.DeleteRequest{Key: key, Type: dataT})
	returnSt = delRes.GetStatus()
	returnErr = err
	if err != nil {
		// log.Printf("Delete operation failed: %v", err)
	} else {
		switch returnSt {
		case utils.KVDeleted:
			// log.Println("Delete operation completed successfully.")
		case utils.KeyNotFound:
			returnErr = fmt.Errorf("delete failed: could not find key: %s", key)
		case utils.UnknownError:
			returnErr = fmt.Errorf("delete operation failed: %v", err)
		}
	}
	return returnSt, returnErr
}

func getConnectionToServer() *grpc.ClientConn {
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
	return conn
}

// run with flag -server_addr=localhost:PORT, default is localhost:2379
func main() {
	conn := getConnectionToServer()
	defer conn.Close()
	client := pb.NewFrontendClient(conn)

	fmt.Println(get(client, utils.LocalData, "10"))
	fmt.Println(put(client, utils.LocalData, "10", "val"))
	fmt.Println(get(client, utils.LocalData, "10"))
	fmt.Println(del(client, utils.LocalData, "10"))
	fmt.Println(get(client, utils.LocalData, "10"))
}
