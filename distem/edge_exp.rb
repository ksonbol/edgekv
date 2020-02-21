#!/usr/bin/ruby

require 'distem'
require_relative 'conf'

idx = 0
Distem.client do |dis|
    for i in 1..NUM_GROUPS
        local_endpoints = "" # endpoints needed for etcd client
        global_endpoints = "" # endpoints needed for etcd client
        nodes = Array.new(NUM_SRVR_PER_GROUP)
        edge_cl_ips = Array.new(NUM_SRVR_PER_GROUP)
        edge_gw_ips = Array.new(NUM_SRVR_PER_GROUP)
        global_ips = Array.new(NUM_SRVR_PER_GROUP)
        gateway_node = GATEWAY_VNODES[i-1]
        gateway_addr = dis.viface_info(gateway_node,'if0')['address'].split('/')[0] # if0 for gw-edge comm
        for j in 1..NUM_SRVR_PER_GROUP
            n = SERVER_VNODES[idx]
            nodes[j-1] = n
            idx += 1
            # addr = dis.viface_info(n,'if0')['address'].split('/')[0]
            edge_cl_addr = dis.viface_info(n, 'if0')['address'].split('/')[0]    # if0 for edge-cli comm
            etcd_addr = dis.viface_info(n,'if1')['address'].split('/')[0]        # if1 for etcd-etcd comm
            edge_gw_addr = dis.viface_info(n,'if2')['address'].split('/')[0]     # if2 for edge-gw comm
            global_address = dis.viface_info(n,'ifadm')['address'].split('/')[0] # special interface ifadm
            edge_cl_ips[j-1] = edge_cl_addr
            edge_gw_ips[j-1] = edge_gw_addr
            local_endpoints += "#{etcd_addr}:#{ETCD_LOCAL_CLIENT_PORT},"
            global_endpoints += "#{etcd_addr}:#{ETCD_GLOBAL_CLIENT_PORT},"
            global_ips[j-1] = global_address
        end
        local_endpoints = local_endpoints[0..-2]              # remove the last comma
        global_endpoints = global_endpoints[0..-2]            # remove the last comma

        nodes.each_with_index do |node, index|
            edge_cl_addr = edge_cl_ips[index]
            edge_gw_addr = edge_gw_ips[index]
            global_addr = global_ips[index]
            dis.vnode_execute(node, "mkdir -p #{EDGEKV_PARENT_DIR}")
            out = %x(scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -r -i #{SSH_KEY_PATH} edgekv/ root@#{global_addr}:#{EDGEKV_PARENT_DIR})  # copy edge server files
            if $?.exitstatus != 0
                puts "could not copy edge server code to node #{node}!"
            end
            dis.vnode_execute(node, "pkill go;pkill edge")  # kill any previous instances of go & edge.go
            # compile the code
            dis.vnode_execute(node, "cd #{EDGEKV_PARENT_DIR}/edgekv;/usr/local/go/bin/go build -o bin/edge cmd/edge/*")

            # IMPORTANT: without cd to edgekv folder, go doesnt read mod file and raise errors
            # also, since distem client does not keep state, cd and go run need to be run together
            puts dis.vnode_execute(node,
                # "cd #{EDGEKV_PARENT_DIR}/edgekv;export ENDPOINTS=#{endpoints_str};nohup /usr/local/go/bin/go run #{EDGE_FILE} \
                "cd #{EDGEKV_PARENT_DIR}/edgekv;export LOCAL_ENDPOINTS=#{local_endpoints} \
                GLOBAL_ENDPOINTS=#{global_endpoints};nohup ./#{EDGE_FILE} \
                -hostname=#{edge_cl_addr} -port=#{EDGE_PORT} -hostname_gw=#{edge_gw_addr} -port_gw=#{EDGE_GW_PORT} \
                -gateway_addr=#{gateway_addr}:#{GATEWAY_EDGE_PORT} > /root/etcdlog/edge.log  2>&1 &")
                # &> /root/etcdlog/edge.log &")
            puts "Edge server #{index+1} from group #{i} is running"
        end
    end
    # puts "All edge servers running"
end