#!/usr/bin/ruby

require 'distem'
require_relative 'conf'

# this assumes we need one client per edge group
idx = 0
Distem.client do |dis|
    for i in 1..NUM_GROUPS
        # TODO: should we give the client addresses of all nodes in the cluster or just one?
        edge_node = SERVER_VNODES[idx]
        idx += NUM_SRVR_PER_GROUP
        edge_addr = dis.viface_info(edge_node,'if0')['address'].split('/')[0] # if0 for edge-client comm
        node = CLIENT_VNODES[i-1]
        addr = dis.viface_info(node,'if0')['address'].split('/')[0]
        dis.vnode_execute(node, "mkdir -p #{EDGEKV_PARENT_DIR}")
        # global_addr is in the network distem uses to communicate with all vnodes!
        # use it only for ssh and scp purposes
        # note: must have run distem-bootstrap --enable-admin-network
        global_addr = dis.viface_info(node,'ifadm')['address'].split('/')[0] # special interface ifadm
        %x(scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -r -i #{SSH_KEY_PATH} edgekv/ root@#{global_addr}:#{EDGEKV_PARENT_DIR})  # copy client files
        if $?.exitstatus != 0
            puts "could not copy client code to node #{node}!"
        end
        dis.vnode_execute(node, "pkill go;pkill main")  # kill any previous instances of go & main.go
        # compile the code
        dis.vnode_execute(node, "cd #{EDGEKV_PARENT_DIR}/edgekv;/usr/local/go/bin/go build -o bin/client cmd/client/*")
        # clean the log folder
        dis.vnode_execute(node, "rm -rf /root/etcdlog; mkdir /root/etcdlog") 
        # IMPORTANT: without cd to edgekv folder, go doesnt read mod file and raise errors
        # also, since distem client does not keep state, cd and go run need to be run together
        puts dis.vnode_execute(node,
            "cd #{EDGEKV_PARENT_DIR}/edgekv;nohup ./#{CLI_FILE} \
            -server_addr=#{edge_addr}:#{EDGE_PORT} \
            > /root/etcdlog/client.log 2>&1 &")
        puts "Client #{i} running"
    end
end