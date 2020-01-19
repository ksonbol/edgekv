require 'distem'
require_relative 'conf'

edge_port = 2381
etcd_client_port = 2379

SSH_KEY_PATH = '/home/ksonbol/.ssh/id_rsa'
EDGEKV_PARENT_DIR = '/root/go/src/github.com/ksonbol' 
EDGE_FILE = 'bin/edge'

Distem.client do |cl|
    endpoints_str = "" # endpoints needed for etcd client
    SERVER_VNODES.each_with_index do |node, idx|
        addr = cl.viface_info(node,'if0')['address'].split('/')[0]
        endpoints_str += "#{addr}:#{etcd_client_port},"
    end
    endpoints_str = endpoints_str[0..-2]              # remove the last comma

    SERVER_VNODES.each_with_index do |node, idx|
        addr = cl.viface_info(node,'if0')['address'].split('/')[0]
        cl.vnode_execute(node, "mkdir -p #{EDGEKV_PARENT_DIR}")
        out = %x(scp -r -i #{SSH_KEY_PATH} edgekv/ root@#{addr}:#{EDGEKV_PARENT_DIR})  # copy edge server files
        if $?.exitstatus != 0
            puts "could not copy edge server code to node #{node}!"
        end
        cl.vnode_execute(node, "pkill go;pkill edge")  # kill any previous instances of go & edge.go
        # compile the code
        cl.vnode_execute(node, "cd #{EDGEKV_PARENT_DIR}/edgekv;/usr/local/go/bin/go build -o bin/edge cmd/edge/*")

        # IMPORTANT: without cd to edgekv folder, go doesnt read mod file and raise errors
        # also, since distem client does not keep state, cd and go run need to be run together
        puts cl.vnode_execute(node,
            # "cd #{EDGEKV_PARENT_DIR}/edgekv;export ENDPOINTS=#{endpoints_str};nohup /usr/local/go/bin/go run #{EDGE_FILE} \
            "cd #{EDGEKV_PARENT_DIR}/edgekv;export ENDPOINTS=#{endpoints_str};nohup ./#{EDGE_FILE} \
            -hostname=#{addr} -port=#{edge_port} \
            > /root/etcdlog/edge.log  2>&1 &")
            # &> /root/etcdlog/edge.log &")
        puts "Edge server #{idx+1} running"
    end
    # puts "All edge servers running"
end