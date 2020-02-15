require 'distem'
require_relative 'conf'

idx = 0
Distem.client do |cl|
    for i in 1..NUM_GROUPS
        local_endpoints = "" # endpoints needed for etcd client
        global_endpoints = "" # endpoints needed for etcd client
        nodes = Array.new(NUM_SRVR_PER_GROUP)
        serv_node_ips = Array.new(NUM_SRVR_PER_GROUP)
        gateway_node = GATEWAY_VNODES[i-1]
        gateway_addr = cl.viface_info(gateway_node,'if0')['address'].split('/')[0]
        for j in 1..NUM_SRVR_PER_GROUP
            node = SERVER_VNODES[idx]
            nodes.push(node)
            idx += 1
            addr = cl.viface_info(node,'if0')['address'].split('/')[0]
            serv_node_ips.push(addr)
            local_endpoints += "#{addr}:#{etcd_local_client_port},"
            global_endpoints += "#{addr}:#{etcd_global_client_port},"
        end
        local_endpoints = local_endpoints[0..-2]              # remove the last comma
        global_endpoints = global_endpoints[0..-2]              # remove the last comma

        nodes.each_with_index do |node, index|
            addr = serv_node_ips[index]
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
                "cd #{EDGEKV_PARENT_DIR}/edgekv;export LOCAL_ENDPOINTS=#{local_endpoints} \
                GLOBAL_ENDPOINTS=#{global_endpoints};nohup ./#{EDGE_FILE} \
                -hostname=#{addr} -port=#{edge_port} -gateway_addr=#{gateway_addr}:#{gateway_port} \
                > /root/etcdlog/edge.log  2>&1 &")
                # &> /root/etcdlog/edge.log &")
            puts "Edge server #{index+1} from group #{i+1} is running"
        end
    end
    # puts "All edge servers running"
end