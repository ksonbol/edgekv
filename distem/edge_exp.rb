require 'distem'

server_vnodes = ['etcd-1', 'etcd-2', 'etcd-3', 'etcd-4', 'etcd-5']
edge_port = 2381
SSH_KEY_PATH = '/home/ksonbol/.ssh/id_rsa'
EDGE_FILE = '/root/go/src/edgekv/edge/edge.go'

Distem.client do |cl|
    server_vnodes.each_with_index do |node, idx|
        addr = cl.viface_info(node,'if0')['address'].split('/')[0]
        out = %x(scp -r -i #{SSH_KEY_PATH} edgekv/ root@#{addr}:/root/go/src/)  # copy edge server files
        if $?.exitstatus != 0
            puts "could not copy edge server code to node #{node}!"
        end
        puts cl.vnode_execute(node,
            "nohup /usr/local/go/bin/go run #{EDGE_FILE} -hostname=#{addr} -port=#{edge_port} \
            > /root/etcdlog/edge.log  2>&1 &")
            # &> /root/etcdlog/edge.log &")
        puts "Edge server #{idx+1} running"
    end
    # puts "All edge servers running"
end