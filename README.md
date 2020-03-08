# EdgeKV

## Build
```bash
git clone https://github.com/ksonbol/edgekv.git $GOPATH/src/github.com/ksonbol/edgekv
cd $GOPATH/src/github.com/ksonbol/edgekv
make
```

## Usage

1. Start your etcd cluster. For a local cluster:
 
    `etcd`

2. Start the edge nodes, specifying at least one of the etcd endpoints in the ENDPOINTS environment variable. For exampe, for a local cluster:
    
    `ENDPOINTS="localhost:2379" ./bin/edge`

3. Start the gateway nodes, specifying the associated edge group address, addresses for gateway-to-gateway and gateway-to-edge communication, and the address of the helper node (if not first node in the ring).

   `./bin/gateway -edge_addr=EDGE_ADDR:EDGE_PORT -gateway_addr=ADDRESS:PORT -gateway_edge_addr=ADDRESS:PORT2 for first node`
   
   `./bin/gateway -edge_addr=EDGE_ADDR2:EDGE_PORT2 -gateway_addr=ADDRESS2:PORT -gateway_edge_addr=ADDRESS2:PORT2 -helper_addr=ADDRESS:PORT for other nodes`
   
4. Start the client nodes, passing the edge server address as an argument. For example:
    
    `./bin/client -server_addr=localhost:2381`

Since this is the default value, for a local cluster, you can just run:

    ./bin/client

## Running on Grid'5000 with Distem

controller.rb file is the main entry to the experiments.

To show the usage help:

```bash
cd distem
ruby controller.rb -v
```

```
Usage: ruby controller.rb [options]
    -s, --setup                      perform platform setup
    -r, --reserve                    make node reservations in setup
    -d, --deploy                     deploy filesystem to pnodes in setup
    -p, --play                       start and test etcd and edge servers
    -l, --set-latency ENVIRONMENT    Set latencies for 'cloud' or 'edge' environments
```

### Typical usage
- Setup the system and reserve nodes.

    
    ```bash
    ruby controller.rb -rsd
    ```


    - Alternatively, you can setup the system with already existing reserved nodes (of type deploy).

    
    ```bash
    ruby controller.rb -sd
    ```

- Set the latency for the required environment ("edge" or "cloud")
    ```bash
    ruby controller.rb -l edge
    ```

- Start the etcd and edge servers
    ```bash
    ruby controller.rb -p
    ```

- Change the latency setting to the other environment ("edge" or "cloud")
    ```bash
    ruby controller.rb -l cloud
    ```

- Restart the etcd and edge servers in the new environment
    ```bash
    ruby controller.rb -p
    ```
