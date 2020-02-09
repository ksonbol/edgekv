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
// hostname    = flag.String("hostname", "localhost", "The server hostname or public IP address")
// port        = flag.Int("port", 2381, "The server port")
// tls         = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
// certFile    = flag.String("cert_file", "", "The TLS cert file")
// keyFile     = flag.String("key_file", "", "The TLS key file")
// gatewayAddr = flag.String("gateway_addr", "localhost:5554", "The address of assigned gateway node")
)

// run with flags -hostname=HOSTNAME -port=PORTNO -gateway_addr=ADDR
// default node addr is localhost:2381, gateway addr is localhost:5554
// must have set LOCAL_ENDPOINTS and GLOBAL_ENDPOINTS env variables
// to at least one of etcd endpoints for each cluster
func main2() {
	flag.Parse()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM) // CTRL-C->SIGINT, kill $PID->SIGTERM
	server := edge.NewEdgeServer(*hostname, *port)
	server.SetGateway(*gatewayAddr)
	go server.Run(*tls, *certFile, *keyFile)
	fmt.Println("Listening to client requests")
	<-sigs
	(*server).Close()
	fmt.Println("Stopping the server...")
	os.Exit(0)
}
