#!/usr/bin/ruby

require 'distem'
require_relative 'conf'

Distem.client do |dis|
    SERVER_VNODES.each_with_index do |node, index|
        addr = dis.viface_info(node,'if1')['address'].split('/')[0]
        dis.vnode_execute(node, "etcdctl --endpoints=#{addr}:#{ETCD_LOCAL_CLIENT_PORT} del \"\" --prefix")
        dis.vnode_execute(node, "etcdctl --endpoints=#{addr}:#{ETCD_GLOBAL_CLIENT_PORT} del \"\" --prefix")
    end
end