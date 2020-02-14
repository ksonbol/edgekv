package main

import (
	"flag"
	"fmt"

	"github.com/ksonbol/edgekv/pkg/client"
	"github.com/ksonbol/edgekv/utils"
)

var (
	serverAddr         = flag.String("server_addr", "localhost:2381", "The server address in the format of host:port")
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name used to verify the hostname returned by TLS handshake")
)

// run with flag -server_addr=localhost:PORT, default is localhost:2381
func main() {
	flag.Parse()
	cl, err := client.NewEdgekvClient(*serverAddr, *tls, *caFile, *serverHostOverride)
	if err != nil {
		fmt.Printf("Error while creating edgekv client %v\n", err)
	}
	key := "key"
	fmt.Println("Testing Local data access")
	fmt.Println(cl.Get(key, utils.LocalDataStr, false))
	fmt.Println(cl.Put(key, utils.LocalDataStr, "val", false))
	fmt.Println(cl.Get(key, utils.LocalDataStr, false))
	fmt.Println(cl.Del(key, utils.LocalDataStr, false))
	fmt.Println(cl.Get(key, utils.LocalDataStr, false))
	fmt.Println("Testing Global data access")
	fmt.Println(cl.Get(key, utils.GlobalDataStr, false))
	fmt.Println(cl.Put(key, utils.GlobalDataStr, "val2", false))
	fmt.Println(cl.Get(key, utils.GlobalDataStr, false))
	fmt.Println(cl.Get(key, utils.LocalDataStr, false))
	fmt.Println(cl.Del(key, utils.GlobalDataStr, false))
	fmt.Println(cl.Get(key, utils.GlobalDataStr, false))
}
