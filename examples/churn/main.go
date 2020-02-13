package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ksonbol/edgekv/pkg/client"
	"github.com/ksonbol/edgekv/pkg/dht"
	"github.com/ksonbol/edgekv/pkg/edge"
	"github.com/ksonbol/edgekv/pkg/gateway"
	"github.com/ksonbol/edgekv/utils"
)

var (
	hostname           = flag.String("hostname", "localhost", "The server hostname or public IP address")
	port               = flag.Int("port", 2381, "The server port")
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile           = flag.String("cert_file", "", "The TLS cert file")
	keyFile            = flag.String("key_file", "", "The TLS key file")
	gatewayAddr        = flag.String("gateway_addr", "localhost:5554", "The address of assigned gateway node")
	gateway2Addr       = flag.String("gateway_addr2", "localhost:5555", "The address of assigned gateway node")
	gateway3Addr       = flag.String("gateway_addr3", "localhost:5556", "The address of assigned gateway node")
	edgeAddr           = flag.String("edge_addr", "localhost:2381", "The server address in the format of host:port")
	edgeAddr2          = flag.String("edge2_addr", "localhost:2384", "The server address in the format of host:port")
	edgeAddr3          = flag.String("edge3_addr", "localhost:2387", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name used to verify the hostname returned by TLS handshake")
)

func main() {
	flag.Parse()
	fmt.Println("Starting the dynamic example")
	fmt.Println("Starting edge groups")
	// Change etcd endpoints for each group
	os.Setenv("LOCAL_ENDPOINTS", "127.0.0.1:2479")
	os.Setenv("GLOBAL_ENDPOINTS", "127.0.0.1:2379")
	time.Sleep(1 * time.Second)
	server := edge.NewEdgeServer(*hostname, *port)
	server2 := edge.NewEdgeServer(*hostname, *port+1)
	server3 := edge.NewEdgeServer(*hostname, *port+2)
	os.Setenv("LOCAL_ENDPOINTS", "127.0.0.1:3479")
	os.Setenv("GLOBAL_ENDPOINTS", "127.0.0.1:3379")
	time.Sleep(1 * time.Second)
	server4 := edge.NewEdgeServer(*hostname, *port+3)
	server5 := edge.NewEdgeServer(*hostname, *port+4)
	server6 := edge.NewEdgeServer(*hostname, *port+5)
	time.Sleep(1 * time.Second)
	os.Setenv("LOCAL_ENDPOINTS", "127.0.0.1:4479")
	os.Setenv("GLOBAL_ENDPOINTS", "127.0.0.1:4379")
	server7 := edge.NewEdgeServer(*hostname, *port+6)
	server8 := edge.NewEdgeServer(*hostname, *port+7)
	server9 := edge.NewEdgeServer(*hostname, *port+8)
	fmt.Println("Setting gateway addresses in edge groups")
	server.SetGateway(*gatewayAddr)
	server2.SetGateway(*gatewayAddr)
	server3.SetGateway(*gatewayAddr)
	server4.SetGateway(*gateway2Addr)
	server5.SetGateway(*gateway2Addr)
	server6.SetGateway(*gateway2Addr)
	server7.SetGateway(*gateway3Addr)
	server8.SetGateway(*gateway3Addr)
	server9.SetGateway(*gateway3Addr)
	fmt.Println("Running the edge groups servers")
	go server.Run(*tls, *certFile, *keyFile)
	go server2.Run(*tls, *certFile, *keyFile)
	go server3.Run(*tls, *certFile, *keyFile)
	go server4.Run(*tls, *certFile, *keyFile)
	go server5.Run(*tls, *certFile, *keyFile)
	go server6.Run(*tls, *certFile, *keyFile)
	go server7.Run(*tls, *certFile, *keyFile)
	go server8.Run(*tls, *certFile, *keyFile)
	go server9.Run(*tls, *certFile, *keyFile)

	fmt.Println("Starting dht nodes")
	var err error
	var node, node2, node3 *dht.Node
	var cli *client.EdgekvClient
	var ncli, ncli2, ncli3 *client.EdgekvClient
	ncli, err = client.NewEdgekvClient(*edgeAddr, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	// fmt.Println("Connection to edge group 1 established")
	st1 := gateway.NewStorage(ncli)
	node = dht.NewLocalNode(*gatewayAddr, st1, nil)

	ncli2, err = client.NewEdgekvClient(*edgeAddr2, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	// fmt.Println("Connection to edge group 2 established")
	st2 := gateway.NewStorage(ncli2)

	ncli3, err = client.NewEdgekvClient(*edgeAddr3, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	// fmt.Println("Connection to edge group 3 established")
	st3 := gateway.NewStorage(ncli3)

	fmt.Println("Created DHT nodes")
	node.Join(nil)
	fmt.Println("Node 1 has joined the DHT")
	fmt.Printf("Node 1 IDs: %s\n", node.ID)
	// fmt.Printf("Node IDs are: %s %s %s\n", node.ID, node2.ID, node3.ID)

	fmt.Println("Creating end client")
	cli, err = client.NewEdgekvClient(*edgeAddr, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}

	time.Sleep(5 * time.Second) // wait for dht to stabilize
	// testing storage
	fmt.Println("Putting keys in global storage")
	fmt.Println(cli.Put("k", utils.GlobalDataStr, "global"))
	fmt.Println(cli.Put("k2", utils.GlobalDataStr, "global2"))
	fmt.Println(cli.Put("k3", utils.GlobalDataStr, "global3"))
	fmt.Println(cli.Put("k4", utils.GlobalDataStr, "global4"))
	fmt.Println(cli.Put("k5", utils.GlobalDataStr, "global5"))
	fmt.Println(cli.Put("k6", utils.GlobalDataStr, "global6"))

	for {
		var cmd string
		var num int
		fmt.Println("Enter command: 'j' for join or 'l' for leave")
		fmt.Scan(&cmd)
		fmt.Println("Enter the number of node to join/leave the dht: [1-3]")
		fmt.Scan(&num)
		switch cmd {
		case "j":
			switch num {
			case 2:
				node2 = dht.NewLocalNode(*gateway2Addr, st2, nil)
				helperNode2 := dht.NewRemoteNode(node.Addr, node.ID, node2.Transport, nil)
				time.Sleep(time.Second)
				node2.Join(helperNode2)
				fmt.Printf("Node 2 IDs: %s\n", node2.ID)
				fmt.Println("Node 2 joined")
			case 3:
				node3 = dht.NewLocalNode(*gateway3Addr, st3, nil)
				helperNode3 := dht.NewRemoteNode(node.Addr, node.ID, node3.Transport, nil)
				time.Sleep(time.Second)
				node3.Join(helperNode3)
				fmt.Printf("Node 3 IDs: %s\n", node3.ID)
				time.Sleep(2 * time.Second)
				fmt.Println("Node 3 joined")
			}
		case "l":
			switch num {
			case 1:
				node.Leave()
				time.Sleep(2 * time.Second)
				fmt.Println("Node 1 leaved")
			case 2:
				node2.Leave()
				time.Sleep(2 * time.Second)
				fmt.Println("Node 2 leaved")
			case 3:
				node3.Leave()
				time.Sleep(2 * time.Second)
				fmt.Println("Node 3 leaved")
			}
		}
	}
}
