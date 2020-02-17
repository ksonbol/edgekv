#!/usr/bin/ruby -w

gem 'ruby-cute', ">=0.6"
require 'cute'
require 'pp' # pretty print
require 'distem'
require_relative 'conf'

SITE_NAME = "nancy"
HOSTNAME = "g5k"

g5k = Cute::G5K::API.new(:username => "ksonbol")
jobs = g5k.get_my_jobs(SITE_NAME)
raise "No jobs running! Run ruby platform_setup.rb --reserve to create a job" unless jobs.length() > 0

if jobs.length() > 1
  puts "WARNING: You have multiple jobs running at #{SITE_NAME}"
end

job = jobs.first

pnodes = job['assigned_nodes']
pnodes.map!{|n| n.split(".")[0]}  # remove the ".SITE_NAME.grid5000.fr" suffix
raise 'This experiment requires at least two physical machines' unless pnodes.size >= 2

local_cluster_token = "local-cluster"
global_cluster_token = "global-cluster"

# these values works fine for both cloud and edge (for now)
hb_interval = 10   # heartbeat interval in ms
elec_timeout = 100 # election timeout in ms

# case SETUP
# when "cloud"
#     hb_interval = 10   # heartbeat interval in ms
#     elec_timeout = 100 # election timeout in ms
# when "edge"
#     hb_interval = 15   # heartbeat interval (1.5xRTT) in ms
#     elec_timeout = 150 # election timeout in ms
# end

idx = 0
Distem.client do |dis|
    for i in 1..NUM_GROUPS
        serv_node_ips = Array.new(NUM_SRVR_PER_GROUP)
        nodes = Array.new(NUM_SRVR_PER_GROUP)
        initial_cluster_local = "" # needed for etcd peers (servers)
        initial_cluster_global = "" # needed for etcd peers (servers)
        # get node IPs and prepare initial cluster conf
        for j in 1..NUM_SRVR_PER_GROUP
            n = SERVER_VNODES[idx]
            nodes[j-1] = n
            idx += 1
            addr = dis.viface_info(n,'if0')['address'].split('/')[0]
            serv_node_ips[j-1] = addr
            initial_cluster_local += "#{n}-local=http://#{addr}:#{ETCD_LOCAL_PEER_PORT},"
            initial_cluster_global += "#{n}-global=http://#{addr}:#{ETCD_GLOBAL_PEER_PORT},"
            # if this is already in the fs, no need to export ETCDCTL_API=3
            dis.vnode_execute(n, "pkill etcd")  # kill any previous instances of etcd
        end
        initial_cluster_local = initial_cluster_local[0..-2]  # remove the last comma
        initial_cluster_global = initial_cluster_global[0..-2]  # remove the last comma
        sleep(5)  # make sure old etcd instances are dead
        nodes.each_with_index do |node, index|
            # clean the log folder
            dis.vnode_execute(node, "rm -rf /root/etcdlog /root/*.etcd; mkdir /root/etcdlog") 
            addr = serv_node_ips[index]

            # local etcd instance
            cmd =  "cd /root;nohup /usr/local/bin/etcd --heartbeat-interval=#{hb_interval} \
            --election-timeout=#{elec_timeout} \
            --name #{node}-local --initial-advertise-peer-urls http://#{addr}:#{ETCD_LOCAL_PEER_PORT} \
            --listen-peer-urls http://#{addr}:#{ETCD_LOCAL_PEER_PORT} \
            --listen-client-urls http://#{addr}:#{ETCD_LOCAL_CLIENT_PORT},http://127.0.0.1:#{ETCD_LOCAL_CLIENT_PORT} \
            --advertise-client-urls http://#{addr}:#{ETCD_LOCAL_CLIENT_PORT} \
            --initial-cluster-token #{local_cluster_token}-#{i} \
            --initial-cluster #{initial_cluster_local} \
            --initial-cluster-state new > /root/etcdlog/etcd_local.log 2>&1 &"
            # --initial-cluster-state new &> /root/etcdlog/out.log &"
            # IMPORTANT: without the last part of the command the function blocks forever!
            # should we add this to listen-client-urls? ,http://127.0.0.1:4001
            puts dis.vnode_execute(node, cmd)
            # puts "etcd server #{idx+1} running"

            # global etcd instance
            cmd =  "cd /root;nohup /usr/local/bin/etcd --heartbeat-interval=#{hb_interval} \
            --election-timeout=#{elec_timeout} \
            --name #{node}-global --initial-advertise-peer-urls http://#{addr}:#{ETCD_GLOBAL_PEER_PORT} \
            --listen-peer-urls http://#{addr}:#{ETCD_GLOBAL_PEER_PORT} \
            --listen-client-urls http://#{addr}:#{ETCD_GLOBAL_CLIENT_PORT},http://127.0.0.1:#{ETCD_GLOBAL_CLIENT_PORT} \
            --advertise-client-urls http://#{addr}:#{ETCD_GLOBAL_CLIENT_PORT} \
            --initial-cluster-token #{global_cluster_token}-#{i} \
            --initial-cluster #{initial_cluster_global} \
            --initial-cluster-state new > /root/etcdlog/etcd_global.log 2>&1 &"
            # --initial-cluster-state new &> /root/etcdlog/out.log &"
            # IMPORTANT: without the last part of the command the function blocks forever!
            # should we add this to listen-client-urls? ,http://127.0.0.1:4001
            puts dis.vnode_execute(node, cmd)
            # puts "etcd server #{idx+1} running"
        end
        sleep(7)
        dis.vnode_execute(nodes[0], "etcdctl --endpoints=#{serv_node_ips[0]}:#{ETCD_LOCAL_CLIENT_PORT} put mykey local")
        dis.vnode_execute(nodes[0], "etcdctl --endpoints=#{serv_node_ips[0]}:#{ETCD_GLOBAL_CLIENT_PORT} put mykey global")
        sleep(3)
        out = dis.vnode_execute(nodes[1], "etcdctl --endpoints=#{serv_node_ips[1]}:#{ETCD_LOCAL_CLIENT_PORT} get mykey")
        out2 = dis.vnode_execute(nodes[1], "etcdctl --endpoints=#{serv_node_ips[1]}:#{ETCD_GLOBAL_CLIENT_PORT} get mykey")
        if out.length>=2 && out[1] == "local" && out2.length >= 2 && out2[1] == "global"
            puts "etcd cluster-#{i} is working correctly"
        else
            puts "etcd cluster-#{i} not setup correctly: '#{out} #{out2}'"
        end
        dis.vnode_execute(nodes[2], "etcdctl --endpoints=#{nodes[2]}:#{ETCD_LOCAL_CLIENT_PORT} del mykey")
        dis.vnode_execute(nodes[2], "etcdctl --endpoints=#{nodes[2]}:#{ETCD_GLOBAL_CLIENT_PORT} del mykey")
    end
end
# puts "all etcd servers are now running!"