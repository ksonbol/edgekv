#!/usr/bin/ruby

require 'distem'
require_relative 'conf'

Distem.client do |cl|
    CLIENT_VNODES.each_with_index do |node, idx|
        # use this addr for scp or ssh only
        global_addr = cl.viface_info(node,'ifadm')['address'].split('/')[0] # special interface ifadm
        # copy go-ycsb repo
        cl.vnode_execute(node, "mkdir -p #{YCSB_PARENT_DIR}")
        %x(scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -r -i #{SSH_KEY_PATH} go-ycsb/ root@#{global_addr}:#{YCSB_PARENT_DIR})
        if $?.exitstatus != 0
            puts "could not copy ycsb code to node #{node}!"
        end
        cl.vnode_execute(node, "bash -c 'cd #{YCSB_PARENT_DIR}/go-ycsb;make'")  # compile go-ycsb code
    end
end