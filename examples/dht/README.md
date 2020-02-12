# Running DHT Nodes (without storage)

main.go file when run without arguments creates four local dht nodes on ports 5554 to 5557

Addresses can be changed as follows:

```$ go run main.go -node_addr=localhost:5550 -node2_addr=localhost:6000```