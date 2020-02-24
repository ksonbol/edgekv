#!/usr/bin/ruby

require 'distem'
require_relative 'conf'

num_gw = 1000
num_cli = 1

gw_nodes = []
(1..num_gw).each do |i|
    gw_nodes << "gw-#{i}"
end

cli_nodes = []
(1..num_cli).each do |i|
    cli_nodes << "cli-#{i}"
end

nodes = gw_nodes + cli_nodes

helper_addr = ""

Distem.client do |dis|
    gw_nodes.each_with_index do |node, index|
        addr = dis.viface_info(node,'if0')['address'].split('/')[0] # if1 for gw-gw comm
        global_address = dis.viface_info(node,'ifadm')['address'].split('/')[0] # special interface ifadm
        if index == 0
            helper_addr = addr # use first node as helper node for all other nodes (this is set once only)
        end
        out = %x(scp -q -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -r -i #{SSH_KEY_PATH} edgekv/ root@#{global_address}:#{EDGEKV_PARENT_DIR})  # copy gateway files
        if $?.exitstatus != 0
            puts "could not copy gateway code to node #{node}!"
        end
        dis.vnode_execute(node, "pkill go;pkill gateway")  # kill any previous instances of go & gateway.go
        # compile the code
        dis.vnode_execute(node,
            "cd #{EDGEKV_PARENT_DIR}/edgekv;/usr/local/go/bin/go build -o bin/gateway cmd/gateway/*")
        if index == 0
            puts dis.vnode_execute(node,
                "cd #{EDGEKV_PARENT_DIR}/edgekv;nohup ./#{GATEWAY_FILE} -gateway_addr=#{addr}:#{GATEWAY_PORT} \
                > /root/etcdlog/gateway.log  2>&1 &")
            sleep(3) # seconds
        else
            puts dis.vnode_execute(node,
                "cd #{EDGEKV_PARENT_DIR}/edgekv;nohup ./#{GATEWAY_FILE} -gateway_addr=#{addr}:#{GATEWAY_PORT} \
                -helper_addr=#{helper_addr}:#{GATEWAY_PORT} > /root/etcdlog/gateway.log  2>&1 &")
        end
        puts "Gateway server #{index+1} is running"
    end
end