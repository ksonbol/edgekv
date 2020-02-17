require 'distem'
require_relative 'conf'

idx = 0
Distem.client do |dis|
    for i in 1..NUM_GROUPS
        local_endpoints = "" # endpoints needed for etcd client
        global_endpoints = "" # endpoints needed for etcd client
        nodes = Array.new(NUM_SRVR_PER_GROUP)
        serv_node_ips = Array.new(NUM_SRVR_PER_GROUP)
        gateway_node = GATEWAY_VNODES[i-1]
        gateway_addr = dis.viface_info(gateway_node,'if0')['address'].split('/')[0]
        for j in 1..NUM_SRVR_PER_GROUP
            n = SERVER_VNODES[idx]
            nodes[j-1] = n
            idx += 1
            addr = dis.viface_info(n,'if0')['address'].split('/')[0]
            serv_node_ips[j-1] = addr
            local_endpoints += "#{addr}:#{ETCD_LOCAL_CLIENT_PORT},"
            global_endpoints += "#{addr}:#{ETCD_GLOBAL_CLIENT_PORT},"
        end
        local_endpoints = local_endpoints[0..-2]              # remove the last comma
        global_endpoints = global_endpoints[0..-2]              # remove the last comma

        nodes.each_with_index do |node, index|
            addr = serv_node_ips[index]
            dis.vnode_execute(node, "mkdir -p #{EDGEKV_PARENT_DIR}")
            out = %x(scp -r -i #{SSH_KEY_PATH} edgekv/ root@#{addr}:#{EDGEKV_PARENT_DIR})  # copy edge server files
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
                -hostname=#{addr} -port=#{EDGE_PORT} -gateway_addr=#{gateway_addr}:#{GATEWAY_PORT} \
                > /root/etcdlog/edge.log  2>&1 &")
                # &> /root/etcdlog/edge.log &")
            puts "Edge server #{index+1} from group #{i} is running"
        end
    end
    # puts "All edge servers running"
end