require 'distem'

server_vnodes = ['etcd-1', 'etcd-2', 'etcd-3', 'etcd-4', 'etcd-5']
client_vnodes = ['cli-1']
edge_port = 2381
SSH_KEY_PATH = '/home/ksonbol/.ssh/id_rsa'
CLI_FILE = '/root/go/src/edgekv/client/client.go'

Distem.client do |cl|
    # TODO: should we give the client addresses of all nodes in the cluster or just one?
    edge_addr = cl.viface_info(server_vnodes[0],'if0')['address'].split('/')[0]
    client_vnodes.each_with_index do |node, idx|
        addr = cl.viface_info(node,'if0')['address'].split('/')[0]
        out = %x(scp -r -i #{SSH_KEY_PATH} edgekv/ root@#{addr}:/root/go/src/)  # copy edge client files
        if $?.exitstatus != 0
            puts "could not copy edge client code to node #{node}!"
        end
        puts cl.vnode_execute(node,
            "nohup /usr/local/go/bin/go run #{CLI_FILE} -server_addr=#{edge_addr}:#{edge_port} \
            > /root/etcdlog/client.log 2>&1 &")
        puts "Client #{idx+1} running"
    end
end