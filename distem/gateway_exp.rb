require 'distem'
require_relative 'conf'

edge_idx = 0
helper_addr = ""
Distem.client do |dis|
    GATEWAY_VNODES.each_with_index do |node, index|
        edge_node = SERVER_VNODES[edge_idx] # we connect the gateway to one of the edge nodes in the group
        edge_idx += NUM_SRVR_PER_GROUP
        edge_addr = dis.viface_info(edge_node,'if0')['address'].split('/')[0] 
        dis.vnode_execute(node, "mkdir -p #{EDGEKV_PARENT_DIR}")
        addr = dis.viface_info(node,'if0')['address'].split('/')[0]
        if index == 0
            helper_addr = addr
        end
        out = %x(scp -r -i #{SSH_KEY_PATH} edgekv/ root@#{addr}:#{EDGEKV_PARENT_DIR})  # copy gateway files
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
                -edge_addr=#{edge_addr}:#{EDGE_PORT} > /root/etcdlog/gateway.log  2>&1 &")
            sleep(3) # seconds
        else
            puts dis.vnode_execute(node,
                "cd #{EDGEKV_PARENT_DIR}/edgekv;nohup ./#{GATEWAY_FILE} -gateway_addr=#{addr}:#{GATEWAY_PORT} \
                -edge_addr=#{edge_addr}:#{EDGE_PORT} -helper_addr=#{helper_addr}:#{GATEWAY_PORT} \
                > /root/etcdlog/gateway.log  2>&1 &")
        end
        puts "Gateway server #{index+1} is running"
    end
end
