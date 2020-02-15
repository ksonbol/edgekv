require 'distem'
require_relative 'conf'

edge_idx = 0
helper_addr = ""
Distem.client do |cl|
    GATEWAY_VNODES.each_with_index do |node, index|
        edge_node = SERVER_VNODES[edge_idx] # we connect the gateway to one of the edge nodes in the group
        edge_idx += NUM_SRVR_PER_GROUP
        edge_addr = cl.viface_info(edge_addr,'if0')['address'].split('/')[0] 
        cl.vnode_execute(node, "mkdir -p #{EDGEKV_PARENT_DIR}")
        addr = cl.viface_info(node,'if0')['address'].split('/')[0]
        if index == 0
            helper_addr = addr
        end
        out = %x(scp -r -i #{SSH_KEY_PATH} edgekv/ root@#{addr}:#{EDGEKV_PARENT_DIR})  # copy gateway files
        if $?.exitstatus != 0
            puts "could not copy gateway code to node #{node}!"
        end
        cl.vnode_execute(node, "pkill go;pkill gateway")  # kill any previous instances of go & gateway.go
        # compile the code
        cl.vnode_execute(node,
            "cd #{EDGEKV_PARENT_DIR}/edgekv;/usr/local/go/bin/go build -o bin/gateway cmd/gateway/*")
        if index == 0
            puts cl.vnode_execute(node,
                "cd #{EDGEKV_PARENT_DIR}/edgekv;nohup ./#{GATEWAY_FILE} -gateway_addr=#{addr}:#{gateway_port} \
                -edge_addr=#{edge_addr}:#{edge_port} > /root/etcdlog/gateway.log  2>&1 &")
            sleep(3) # seconds
        else
            puts cl.vnode_execute(node,
                "cd #{EDGEKV_PARENT_DIR}/edgekv;nohup ./#{GATEWAY_FILE} -gateway_addr=#{addr}:#{gateway_port} \
                -edge_addr=#{edge_addr}:#{edge_port} -helper_addr=#{helper_addr}:#{gateway_port} \
                > /root/etcdlog/gateway.log  2>&1 &")
        end
        puts "Gateway server #{index+1} is running"
    end
end
