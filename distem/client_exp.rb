require 'distem'
require_relative 'conf'

SSH_KEY_PATH = '/home/ksonbol/.ssh/id_rsa'

# this assumes we need one client per edge group
idx = 0
Distem.client do |cl|
    for i in 1..NUM_GROUPS
        # TODO: should we give the client addresses of all nodes in the cluster or just one?
        edge_node = SERVER_VNODES[idx]
        idx += NUM_SRVR_PER_GROUP
        edge_addr = cl.viface_info(edge_node,'if0')['address'].split('/')[0]
        node = CLIENT_VNODES[i]
        addr = cl.viface_info(node,'if0')['address'].split('/')[0]
        cl.vnode_execute(node, "mkdir -p #{EDGEKV_PARENT_DIR}")
        system("scp -r -i #{SSH_KEY_PATH} edgekv/ root@#{addr}:#{EDGEKV_PARENT_DIR}")  # copy client files
        if $?.exitstatus != 0
            puts "could not copy client code to node #{node}!"
        end
        cl.vnode_execute(node, "pkill go;pkill main")  # kill any previous instances of go & main.go
        # compile the code
        cl.vnode_execute(node, "cd #{EDGEKV_PARENT_DIR}/edgekv;/usr/local/go/bin/go build -o bin/client cmd/client/*")
        # clean the log folder
        cl.vnode_execute(node, "rm -rf /root/etcdlog; mkdir /root/etcdlog") 
        # IMPORTANT: without cd to edgekv folder, go doesnt read mod file and raise errors
        # also, since distem client does not keep state, cd and go run need to be run together
        puts cl.vnode_execute(node,
            "cd #{EDGEKV_PARENT_DIR}/edgekv;nohup ./#{CLI_FILE} \
            -server_addr=#{edge_addr}:#{edge_port} \
            > /root/etcdlog/client.log 2>&1 &")
        puts "Client #{idx+1} running"
    end
end