#!/usr/bin/ruby -w

require 'distem'
require_relative 'conf'

workload = "workloada"
local_prop = 0.5
exp_num = ARGV[0].to_i

case exp_num
when 1
    local_prop = 1.0
when 2
    local_prop = 0.2
when 3
    local_prop = 0.8
end

edge_idx = 0
edge_addr = Array.new(NUM_SRVR_PER_GROUP)

Distem.client do |cl|
    # get edge addresses
    CLIENT_VNODES.each_with_index do |node, idx|
        # use this address for scp or ssh only
        global_addr = cl.viface_info(node,'ifadm')['address'].split('/')[0] # special interface ifadm
        system("scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i #{SSH_KEY_PATH} edgekv.conf root@#{global_addr}:/root")
        edge_node = SERVER_VNODES[edge_idx]
        edge_idx += NUM_SRVR_PER_GROUP
        e_addr =  cl.viface_info(edge_node,'if0')['address'].split('/')[0] # if0 for edge-client comm
        edge_addr[idx] = "#{e_addr}:#{EDGE_PORT}"
    end
    # load data into all clients
    CLIENT_VNODES.each_with_index do |node, idx|
        cl.vnode_execute(node, "cd #{YCSB_PARENT_DIR}/go-ycsb;nohup \
            ./bin/go-ycsb load edgekv -P workloads/#{workload} -P  /root/edgekv.conf \
            -p edgekv.serverAddr=#{edge_addr[idx]} -p edgekv.localproportion=#{local_prop} \
             > /root/etcdlog/ycsb-load-#{exp_num}.log  2>&1 &"
        )
    end
    # run the workloads for all clients at the same time
    CLIENT_VNODES.each_with_index do |node, idx|
            cl.vnode_execute(node, "cd #{YCSB_PARENT_DIR}/go-ycsb;nohup \
                ./bin/go-ycsb run edgekv -P workloads/#{workload} -P  /root/edgekv.conf \
                -p edgekv.serverAddr=#{edge_addr[idx]} -p edgekv.localproportion=#{local_prop} \
                > /root/etcdlog/ycsb-run-#{exp_num}.log  2>&1 &"
            )
    end
end