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
	port               = flag.Int("port", 2385, "The server port")
	hostnameGw         = flag.String("hostnameGw", "localhost", "The server hostname or public IP address")
	portGw             = flag.Int("portGw", 2395, "The server port")
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile           = flag.String("cert_file", "", "The TLS cert file")
	keyFile            = flag.String("key_file", "", "The TLS key file")
	gatewayAddr        = flag.String("gateway_addr", "localhost:5554", "The address of assigned gateway node")
	gateway2Addr       = flag.String("gateway_addr2", "localhost:5555", "The address of assigned gateway node")
	gateway3Addr       = flag.String("gateway_addr3", "localhost:5556", "The address of assigned gateway node")
	gatewayAddrEdge    = flag.String("gateway_edge_addr", "localhost:5564", "The address of assigned gateway node")
	gateway2AddrEdge   = flag.String("gateway_edge_addr2", "localhost:5565", "The address of assigned gateway node")
	gateway3AddrEdge   = flag.String("gateway_edge_addr3", "localhost:5566", "The address of assigned gateway node")
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
	server := edge.NewEdgeServer(*hostname, *port, *hostnameGw, *portGw)
	server2 := edge.NewEdgeServer(*hostname, *port+1, *hostnameGw, *portGw+1)
	server3 := edge.NewEdgeServer(*hostname, *port+2, *hostnameGw, *portGw+2)
	os.Setenv("LOCAL_ENDPOINTS", "127.0.0.1:3479")
	os.Setenv("GLOBAL_ENDPOINTS", "127.0.0.1:3379")
	time.Sleep(1 * time.Second)
	server4 := edge.NewEdgeServer(*hostname, *port+3, *hostnameGw, *portGw+3)
	server5 := edge.NewEdgeServer(*hostname, *port+4, *hostnameGw, *portGw+4)
	server6 := edge.NewEdgeServer(*hostname, *port+5, *hostnameGw, *portGw+5)
	time.Sleep(1 * time.Second)
	os.Setenv("LOCAL_ENDPOINTS", "127.0.0.1:4479")
	os.Setenv("GLOBAL_ENDPOINTS", "127.0.0.1:4379")
	server7 := edge.NewEdgeServer(*hostname, *port+6, *hostnameGw, *portGw+6)
	server8 := edge.NewEdgeServer(*hostname, *port+7, *hostnameGw, *portGw+7)
	server9 := edge.NewEdgeServer(*hostname, *port+8, *hostnameGw, *portGw+8)
	fmt.Println("Setting gateway addresses in edge groups")
	server.SetGateway(*gatewayAddrEdge)
	server2.SetGateway(*gatewayAddrEdge)
	server3.SetGateway(*gatewayAddrEdge)
	server4.SetGateway(*gateway2AddrEdge)
	server5.SetGateway(*gateway2AddrEdge)
	server6.SetGateway(*gateway2AddrEdge)
	server7.SetGateway(*gateway3AddrEdge)
	server8.SetGateway(*gateway3AddrEdge)
	server9.SetGateway(*gateway3AddrEdge)
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
	var cli, cli2, cli3 *client.EdgekvClient
	var ncli, ncli2, ncli3 *client.EdgekvClient
	edgeAddr := fmt.Sprintf("%s:%d", *hostnameGw, *portGw)
	ncli, err = client.NewEdgekvClient(edgeAddr, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	// fmt.Println("Connection to edge group 1 established")
	st1 := gateway.NewStorage(ncli)
	node := dht.NewLocalNode(*gatewayAddr, *gatewayAddrEdge, st1, nil)

	edgeAddr2 := fmt.Sprintf("%s:%d", *hostnameGw, *portGw+3)
	ncli2, err = client.NewEdgekvClient(edgeAddr2, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	// fmt.Println("Connection to edge group 2 established")
	st2 := gateway.NewStorage(ncli2)
	node2 := dht.NewLocalNode(*gateway2Addr, *gateway2AddrEdge, st2, nil)

	edgeAddr3 := fmt.Sprintf("%s:%d", *hostnameGw, *portGw+6)
	ncli3, err = client.NewEdgekvClient(edgeAddr3, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	// fmt.Println("Connection to edge group 3 established")
	st3 := gateway.NewStorage(ncli3)
	node3 := dht.NewLocalNode(*gateway3Addr, *gateway3AddrEdge, st3, nil)

	fmt.Println("Created DHT nodes")
	helperNode2 := dht.NewRemoteNode(node.Addr, node.ID, node2.Transport, nil)
	helperNode3 := dht.NewRemoteNode(node.Addr, node.ID, node3.Transport, nil)
	node.Join(nil)
	node2.Join(helperNode2)
	node3.Join(helperNode3)
	fmt.Println("All nodes have joined the DHT")
	fmt.Printf("Node IDs are: %s %s %s", node.ID, node2.ID, node3.ID)

	fmt.Println("Creating clients")
	edgeClAddr := fmt.Sprintf("%s:%d", *hostname, *port)
	cli, err = client.NewEdgekvClient(edgeClAddr, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	edgeClAddr2 := fmt.Sprintf("%s:%d", *hostname, *port+3)
	cli2, err = client.NewEdgekvClient(edgeClAddr2, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	edgeClAddr3 := fmt.Sprintf("%s:%d", *hostname, *port+6)
	cli3, err = client.NewEdgekvClient(edgeClAddr3, *tls, *certFile, *serverHostOverride)
	if err != nil {
		log.Fatalf("Could not connect to edge group %v", err)
	}
	time.Sleep(5 * time.Second) // wait for dht to stabilize
	// testing storage
	fmt.Println("Testing Local data access")
	cli.Put("k", utils.LocalDataStr, "local", false)
	cli.Put("k2", utils.LocalDataStr, "local2", false)
	cli.Put("k3", utils.LocalDataStr, "local3", false)
	fmt.Println("First group should have local key 'k' while second and third wouldn't")
	fmt.Println(cli.Get("k", utils.LocalDataStr, false))
	fmt.Println(cli2.Get("k", utils.LocalDataStr, false))
	fmt.Println(cli3.Get("k", utils.LocalDataStr, false))
	// fmt.Println(cli.Del(key, utils.LocalDataStr))
	// fmt.Println(cli.Get(key, utils.LocalDataStr))
	fmt.Println("Testing Global data access")
	fmt.Println("Key k should not exist in global data of any group")
	fmt.Println(cli.Get("k", utils.GlobalDataStr, false))
	fmt.Println(cli2.Get("k", utils.GlobalDataStr, false))
	fmt.Println(cli3.Get("k", utils.GlobalDataStr, false))
	fmt.Println("Putting keys in global storage")
	fmt.Println(cli.Put("k", utils.GlobalDataStr, "global", false))
	fmt.Println(cli.Put("k2", utils.GlobalDataStr, "global2", false))
	fmt.Println(cli2.Put("k3", utils.GlobalDataStr, "global3", false))
	fmt.Println(cli2.Put("k4", utils.GlobalDataStr, "global4", false))
	fmt.Println(cli3.Put("k5", utils.GlobalDataStr, "global5", false))
	fmt.Println(cli3.Put("k6", utils.GlobalDataStr, "global6", false))
	fmt.Println("Showing key-value pairs of each edge group")
	fmt.Println(cli.Get("k6", utils.GlobalDataStr, false))
	fmt.Println(cli2.Get("k", utils.GlobalDataStr, false))
	fmt.Println(cli3.Get("k4", utils.GlobalDataStr, false))
	fmt.Println("Local data should not be affected")
	fmt.Println(cli.Get("k", utils.LocalDataStr, false))
	fmt.Println(cli2.Get("k", utils.LocalDataStr, false))
	fmt.Println(cli3.Get("k", utils.LocalDataStr, false))
}
