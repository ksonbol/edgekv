# Running DHT Nodes

main.go file when run without arguments creates four local dht nodes on ports 5554 to 5557

Addresses can be changed as follows:

```$ go run main.go -node_addr=localhost:5550 -node2_addr=localhost:6000```

main_single_node.go file allows running a single local node each time it is run.


To run multiple nodes with it, run the first one without specifying a -helper_addr argument (but you can change the node_addr), and run the rest with specifying any existing node with the -helper_addr argument.

For example:
```
$ go run main_single_node.go

$ go run main_single_node.go -node_addr=localhost:5555 
-helper_addr=localhost:5554

$ go run main_single_node.go -node_addr=localhost:5556 -helper_addr=localhost:5555
```