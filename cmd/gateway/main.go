package main

// This file creates a gateway node: a dht node + connection to edgekv storage

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
	edgeAddress        = flag.String("edge_addr", "localhost:2381", "The server address in the format of host:port")
	gwAddr             = flag.String("gateway_addr", "localhost:5554", "The server address in the format of host:port")
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name used to verify the hostname returned by TLS handshake")
	helperAddr         = flag.String("helper_addr", "", "The helper node address in the format of host:port")
)

// run with flag -edge_addr=localhost:PORT -gateway_addr=localhost:PORT
func main() {
	flag.Parse()
	cli, err := client.NewEdgekvClient(*edgeAddress, *tls, *caFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	fmt.Println("Connection to edge storage established")
	st := gateway.NewStorage(cli)
	gw := dht.NewLocalNode(*gwAddr, st, nil)
	fmt.Println("Edge node created")
	var helperNode *dht.Node
	if *helperAddr != "" {
		helperNode = dht.NewRemoteNode(*helperAddr, "", gw.Transport, nil)
	}
	time.Sleep(2 * time.Second)
	gw.Join(helperNode)
	fmt.Println("Edge node running")
	for {
		time.Sleep(10 * time.Second)
	}
}
