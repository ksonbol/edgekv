NUM_GROUPS = 3
NUM_SRVR_PER_GROUP = 3
NUM_GATEWAYS = NUM_GROUPS
NUM_SERVERS = NUM_GROUPS * NUM_SRVR_PER_GROUP
NUM_CLIENTS = NUM_GROUPS # one client for each edge group
NUM_VNODES = NUM_SERVERS + NUM_GATEWAYS + NUM_CLIENTS

SERVER_VNODES = []
GATEWAY_VNODES = []
CLIENT_VNODES = []

for i in 1..NUM_SERVERS
  SERVER_VNODES.push("edge-#{i}")
end

for i in 1..NUM_GATEWAYS
  GATEWAY_VNODES.push("gw-#{i}")
end

for i in 1..NUM_CLIENTS
  CLIENT_VNODES.push("cli-#{i}")
end

VNODE_LIST = SERVER_VNODES + GATEWAY_VNODES + CLIENT_VNODES

SETUP = "cloud"

etcd_local_client_port = 2379
etcd_global_client_port = 2381
etcd_local_peer_port = 2380
etcd_global_peer_port = 2382

edge_port = 2381

gateway_port = 5554

EDGEKV_PARENT_DIR = '/root/go/src/github.com/ksonbol' 
SSH_KEY_PATH = '/home/ksonbol/.ssh/id_rsa'
EDGE_FILE = 'bin/edge'
GATEWAY_FILE = "bin/gateway"
CLI_FILE = 'bin/client'
