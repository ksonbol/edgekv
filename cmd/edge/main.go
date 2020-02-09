package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ksonbol/edgekv/pkg/edge"
)

var (
	hostname     = flag.String("hostname", "localhost", "The server hostname or public IP address")
	port         = flag.Int("port", 2381, "The server port")
	tls          = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile     = flag.String("cert_file", "", "The TLS cert file")
	keyFile      = flag.String("key_file", "", "The TLS key file")
	gatewayAddr  = flag.String("gateway_addr", "localhost:5554", "The address of assigned gateway node")
	gateway2Addr = flag.String("gateway_addr2", "localhost:5555", "The address of assigned gateway node")
)

// run with flags -hostname=HOSTNAME -port=PORTNO -gateway_addr=ADDR
// default node addr is localhost:2381, gateway addr is localhost:5554
// must have set LOCAL_ENDPOINTS and GLOBAL_ENDPOINTS env variables
// to at least one of etcd endpoints for each cluster
func main() {
	flag.Parse()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM) // CTRL-C->SIGINT, kill $PID->SIGTERM
	fmt.Println("Starting the edge servers")
	server := edge.NewEdgeServer(*hostname, *port)
	server2 := edge.NewEdgeServer(*hostname, *port+1)
	server3 := edge.NewEdgeServer(*hostname, *port+2)
	server4 := edge.NewEdgeServer(*hostname, *port+3)
	server5 := edge.NewEdgeServer(*hostname, *port+4)
	server6 := edge.NewEdgeServer(*hostname, *port+5)
	server.SetGateway(*gatewayAddr)
	server2.SetGateway(*gatewayAddr)
	server3.SetGateway(*gatewayAddr)
	server4.SetGateway(*gateway2Addr)
	server5.SetGateway(*gateway2Addr)
	server6.SetGateway(*gateway2Addr)
	go server.Run(*tls, *certFile, *keyFile)
	go server2.Run(*tls, *certFile, *keyFile)
	go server3.Run(*tls, *certFile, *keyFile)
	go server4.Run(*tls, *certFile, *keyFile)
	go server5.Run(*tls, *certFile, *keyFile)
	go server6.Run(*tls, *certFile, *keyFile)
	fmt.Println("Listening to client requests")
	<-sigs
	fmt.Println("Stopping the servers...")
	(*server).Close()
	(*server2).Close()
	(*server3).Close()
	(*server4).Close()
	(*server5).Close()
	(*server6).Close()
	os.Exit(0)
}
