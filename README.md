<h2>A distributed storage system for the edge</h2>

Running experiments on Distem

controller.rb file is the main entry.
ruby controller.rb -rsdp

Running edge servers
- First, specify the etcd cluster endpoints in the ENDPOINTS environment variable
export ENDPOINTS="IP1:PORT1,IP2:PORT2.."
- Second, run the file:
go run edge/edge.go

Notes:
Add the following to /etc/environment
export ETCDCTL_API=3
* we already added this to our fs image.
