require 'distem'

server_vnodes = ['etcd-1', 'etcd-2', 'etcd-3', 'etcd-4', 'etcd-5']
edge_port = 2381
etcd_client_port = 2379

SSH_KEY_PATH = '/home/ksonbol/.ssh/id_rsa'
EDGEKV_DIR = '/root/go/src/edgekv/' 
EDGE_FILE = 'edge/edge.go'

Distem.client do |cl|
    endpoints_str = "" # endpoints needed for etcd client
    server_vnodes.each_with_index do |node, idx|
        addr = cl.viface_info(node,'if0')['address'].split('/')[0]
        endpoints_str += "#{addr}:#{etcd_client_port},"
    end
    endpoints_str = endpoints_str[0..-2]              # remove the last comma

    server_vnodes.each_with_index do |node, idx|
        addr = cl.viface_info(node,'if0')['address'].split('/')[0]
        out = %x(scp -r -i #{SSH_KEY_PATH} edgekv/ root@#{addr}:/root/go/src/)  # copy edge server files
        if $?.exitstatus != 0
            puts "could not copy edge server code to node #{node}!"
        end
        # IMPORTANT: without cd to edgekv folder, go doesnt read mod file and raise errors
        # also, since distem client does not keep state, cd and go run need to be run together
        puts cl.vnode_execute(node,
            "cd #{EDGEKV_DIR};export ENDPOINTS=#{endpoints_str};nohup /usr/local/go/bin/go run #{EDGE_FILE} \
            -hostname=#{addr} -port=#{edge_port} \
            > /root/etcdlog/edge.log  2>&1 &")
            # &> /root/etcdlog/edge.log &")
        puts "Edge server #{idx+1} running"
    end
    # puts "All edge servers running"
end