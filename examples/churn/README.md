# Testing storage in a stable network
This example tests how the churn in the DHT ring (nodes join and leave) affects the keys. The setup is same as the "dynamic" example.

## Setup:
- 3 gateway (DHT) nodes
- Each node is connected to a 3-node edge group
- Each edge group hosts both a local and a "global" storage
- The storages are etcd clients
- Each edge node is also an etcd node in two clusters (local and global)
- Each node uses different ports for each cluster
- All these nodes are created locally using different ports

The example writes keys to the first edge group only.


## Usage:
First run the Procfiles to setup the 6 etcd clusters (3 local and 3 global ones) with 3 members each.

We will use the Procfiles from the "dynamic" example folder

Install goreman if not already installed: `go get github.com/mattn/goreman`

```
$ cd global
$ goreman -f Procfile start

$ cd local
$ goreman -f Profile start
```

Then, run the example as usual:

`go run examples/dynamic/main.go` (from project root)

To retrieve all keys in a group, use etcdctl as follows (changing the port number based on the cluster):

### Local Clusters:

`$ etcdctl --endpoints="localhost:2479,localhost:22479,localhost:32479"  get "" --prefix`

Should have keys: k, k2, k3

`$ etcdctl --endpoints="localhost:3479,localhost:42479,localhost:52479" get "" --prefix`

Should have no keys

`$ etcdctl --endpoints="localhost:4479,localhost:62479,localhost:63479" get "" --prefix`

Should have no keys


### Global Clusters:

`$ etcdctl --endpoints="localhost:2379,localhost:22379,localhost:32379"  get "" --prefix`

Should have key: 13fbd79c3d390e5d6585a21e11ff5ec1970cff0c (k)

`$ etcdctl --endpoints="localhost:3379,localhost:42379,localhost:52379" get "" --prefix`


Should have (hashed) keys: 

39ef4c0a8f9b9cf16bf85a291d470f9c80482908 (k2)
cb41020ef91a2e7c5369ad8c0e801792d4585583 (k3)
83d81888461c93916c1f8f4af3b1a59f9b4611f9 (k4)
5b94c50a905a67934e8ee420d9096788dcf04ca9 (k5)
346bacdd01f4d124ab2b06e9291b583b88393500 (k6)

`$ etcdctl --endpoints="localhost:4379,localhost:62379,localhost:63379" get "" --prefix`

Should have no keys