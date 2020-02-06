package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/ksonbol/edgekv/pkg/dht"
)

var (
	nodeAddress = flag.String("node_addr", "localhost:5554", "The dht server address in the format of host:port")
	helperAddr  = flag.String("helper_addr", "", "The helper node address in the format of host:port")
)

func run() {
	flag.Parse()
	fmt.Println("Starting  the program")
	node := dht.NewLocalNode(*nodeAddress)
	fmt.Println("Created local nodes")
	var helperNode *dht.Node
	if *helperAddr != "" {
		helperNode = dht.NewRemoteNode(*helperAddr, "", node.Transport)
	}
	time.Sleep(2 * time.Second)
	node.Join(helperNode)
	fmt.Println("The node has joined the system")
	for {
		time.Sleep(5 * time.Second)
		fmt.Printf("Node %s\n", node.Addr)
		fmt.Printf("ID %s\n", node.ID)
		fmt.Printf("Successor: %s, Predecessor: %s\n", node.Successor().ID, node.Predecessor().ID)
		// fmt.Println("Finger table")
		// node.PrintFT()
	}
}
