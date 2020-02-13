# Running a DHT node (without storage)
cmd/dht/main.go file allows running a single local DHT node (without storage) each time it is run.

To run multiple nodes with it, run the first one without specifying a -helper_addr argument (but you can change the node_addr), and run the rest with specifying any existing node as the helper with the -helper_addr argument.

For example:
```
$ go run cmd/dht/main.go

$ go run cmd/dht/main.go -node_addr=localhost:5555 
-helper_addr=localhost:5554

$ go run cmd/dht/main.go -node_addr=localhost:5556 -helper_addr=localhost:5555
```