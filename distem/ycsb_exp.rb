require 'distem'
require_relative 'conf'

# TODO: make this work for each client node
Distem.client do |cl|
    CLIENT_VNODES.each_with_index do |node, idx|
        addr = cl.viface_info(node,'if0')['address'].split('/')[0]
        # copy go-ycsb repo
        cl.vnode_execute(node, "mkdir -p #{YCSB_PARENT_DIR}")
        system("scp -r -i #{SSH_KEY_PATH} go-ycsb/ root@#{addr}:#{YCSB_PARENT_DIR}")
        system("scp -i #{SSH_KEY_PATH} edgekv.conf root@#{addr}:/root")
        if $?.exitstatus != 0
            puts "could not copy ycsb code to node #{node}!"
        end
        cl.vnode_execute(node, "bash -c 'cd #{YCSB_PARENT_DIR}/go-ycsb;make'")  # compile go-ycsb code
        # Run the experiments
    end
end