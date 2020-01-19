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
	hostname = flag.String("hostname", "localhost", "The server hostname or public IP address")
	port     = flag.Int("port", 2381, "The server port")
	tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile = flag.String("cert_file", "", "The TLS cert file")
	keyFile  = flag.String("key_file", "", "The TLS key file")
)

// run with flags -hostname=HOSTNAME -port=PORTNO, default is localhost:2381
// must have set ENDPOINTS env variable to at least one of etcd endpoints
func main() {
	flag.Parse()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM) // CTRL-C->SIGINT, kill $PID->SIGTERM
	server := edge.NewEdgeServer(*hostname, *port)
	go server.Run(*tls, *certFile, *keyFile)
	fmt.Println("Listening to client requests")
	<-sigs
	(*server).Close()
	fmt.Println("Stopping the server...")
	os.Exit(0)
}
