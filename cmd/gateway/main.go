package main

// This file creates a gateway node: a dht node + connection to edgekv storage

import (
	"flag"
	"fmt"
	"time"

	//"github.com/ksonbol/edgekv/pkg/client"
	"github.com/ksonbol/edgekv/pkg/dht"
	"github.com/ksonbol/edgekv/pkg/gateway"
)

var (
	edgeAddress        = flag.String("edge_addr", "localhost:2395", "The server address in the format of host:port")
	gwAddr             = flag.String("gateway_addr", "localhost:5554", "gw addr used by other gw nodes, host:port")
	gwEdgeAddr         = flag.String("gateway_edge_addr", "localhost:5564", "gw addr used by edge nodes, host:port")
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name used to verify the hostname returned by TLS handshake")
	helperAddr         = flag.String("helper_addr", "", "The helper node address in the format of host:port")
)

// run with flag -edge_addr=localhost:PORT -gateway_addr=localhost:PORT -gateway_edge_addr=localhost:PORT2 for first node
// run with flag -edge_addr=localhost:PORT3 -gateway_addr=localhost:PORT4 -gateway_edge_addr=localhost:PORT5 -helper_addr=localhost:PORT2 for other nodes
func main() {
	flag.Parse()
	// cli, err := client.NewEdgekvClient(*edgeAddress, *tls, *caFile, *serverHostOverride)
	// if err != nil {
	// 	log.Fatalf("Could not connect to edge group %v", err)
	// }
	// fmt.Println("Connection to edge storage established")
	// st := gateway.NewStorage(cli)
	st := gateway.NewHashMapStorage()
	gw := dht.NewLocalNode(*gwAddr, *gwEdgeAddr, st, nil)
	fmt.Println("gateway node created")
	var helperNode *dht.Node
	if *helperAddr != "" {
		helperNode = dht.NewRemoteNode(*helperAddr, "", gw.Transport, nil)
	}
	time.Sleep(2 * time.Second)
	gw.Join(helperNode)
	// fmt.Println("gateway node running")
	// fmt.Printf("Node ID: %s\n", gw.ID)
	// var key, val string
	// var err error
	for {
		time.Sleep(10 * time.Second)
		// st.PrintNumKeys()
		// st.PrintKeys()
		// fmt.Println("Enter key to get!")
		// fmt.Scan(&key)
		// val, err = gw.GetKV(key)
		// if err != nil {
		// 	fmt.Printf("Error finding key %s: %v\n", key, err)
		// } else {
		// 	fmt.Printf("Val[%s] = %s\n", key, val)
		// }
	}
}
