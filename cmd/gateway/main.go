package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/ksonbol/edgekv/pkg/client"
	"github.com/ksonbol/edgekv/pkg/dht"
	"github.com/ksonbol/edgekv/pkg/gateway"
)

var (
	nodeAddr           = flag.String("node_addr", "localhost:5554", "The dht server address in the format of host:port")
	node2Addr          = flag.String("node2_addr", "localhost:5555", "The helper node address in the format of host:port")
	edgeAddr           = flag.String("edge_addr", "localhost:2381", "The server address in the format of host:port")
	edgeAddr2          = flag.String("edge2_addr", "localhost:2385", "The server address in the format of host:port")
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name used to verify the hostname returned by TLS handshake")
)

func main() {
	flag.Parse()
	var err error
	var cli, cli2 *client.EdgekvClient
	fmt.Println("Starting the program")
	cli, err = client.NewEdgekvClient(*edgeAddr, *tls, *caFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	fmt.Println("Connection to edge storage established")
	st1 := gateway.NewStorage(cli)
	node := dht.NewLocalNode(*nodeAddr, st1, nil)

	cli2, err = client.NewEdgekvClient(*edgeAddr2, *tls, *caFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	fmt.Println("Connection to edge storage established")
	st2 := gateway.NewStorage(cli2)
	node2 := dht.NewLocalNode(*node2Addr, st2, nil)
	fmt.Println("Created local nodes")
	m := map[int]*dht.Node{1: node, 2: node2}
	helperNode2 := dht.NewRemoteNode(node.Addr, node.ID, node2.Transport, nil)
	time.Sleep(2 * time.Second)
	node.Join(nil)
	node2.Join(helperNode2)
	fmt.Println("All nodes have joined the system")
	fmt.Println("Nodes IDs are: ")
	for k, v := range m {
		fmt.Println(k, v.ID)
	}
	fmt.Println()
	for {
		fmt.Println("Enter the node number (1-2) to print its FT")
		var nodeNum int
		fmt.Scan(&nodeNum)
		n, ok := m[nodeNum]
		if !ok {
			log.Fatalf("Such node does not exist!")
		}
		fmt.Printf("Node %d\n with Addr %s, ID: %s\n", nodeNum, n.Addr, n.ID)
		n.PrintFT()
		fmt.Printf("Successor: %s, Predecessor: %s\n", n.Successor().ID, n.Predecessor().ID)
		time.Sleep(5 * time.Second)
	}
}
