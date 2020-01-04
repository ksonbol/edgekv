#!/usr/bin/ruby -w

gem 'ruby-cute', ">=0.6"
require 'cute'
require 'pp' # pretty print
require 'distem'

SITE_NAME = "nancy"
HOSTNAME = "g5k"

g5k = Cute::G5K::API.new(:username => "ksonbol")
jobs = g5k.get_my_jobs(SITE_NAME)
raise "No jobs running! Run ruby platform_setup.rb --reserve to create a job" unless jobs.length() > 0

if jobs.length() > 1
  puts "WARNING: You have multiple jobs running at #{SITE_NAME}"
end

job = jobs.first
subnet = g5k.get_subnets(job).first
subnet_addr = "#{subnet.address}/#{subnet.prefix}"

pnodes = job['assigned_nodes']
pnodes.map!{|n| n.split(".")[0]}  # remove the ".SITE_NAME.grid5000.fr" suffix
raise 'This experiment requires at least two physical machines' unless pnodes.size >= 2
coordinator = pnodes.first

server_vnodes = ['etcd-1', 'etcd-2', 'etcd-3', 'etcd-4', 'etcd-5']
client_vnodes = ['cli-1']
vnodelist = server_vnodes + client_vnodes
NUM_VNODE = vnodelist.length

instance_name = "edge"
cluster_token = "edge-cluster"

serv_node_ips = Array.new(server_vnodes.length)
initial_cluster_str = "" # needed for etcd peers (servers)
endpoints_str = "" # endpoints needed for etcd client
Distem.client do |dis|

    # update latencies before experiment
    # latency_mat = Array.new(NUM_VNODE) {Array.new(NUM_VNODE, 0)}
    # assume lat(node to itself) = 0
    # assume lat(client to client) = 0
    # dis.set_peers_latencies(vnodelist, latency_mat)
    # sleep(2)


    # get node IPs and prepare initial cluster conf
    server_vnodes.each_with_index do |node, idx|
        addr = dis.viface_info(node,'if0')['address'].split('/')[0]
        serv_node_ips[idx] = addr
        initial_cluster_str += "#{node}=http://#{addr}:2380,"
        endpoints_str += "#{addr}:2379,"
        # if this is already in the fs, no need to export ETCDCTL_API=3
        dis.vnode_execute(node, "pkill etcd")  # kill any previous instances of etcd
    end
    initial_cluster_str = initial_cluster_str[0..-2]  # remove the last comma
    endpoints_str = endpoints_str[0..-2]              # remove the last comma
    sleep(5)  # make sure old etcd instances are dead
    server_vnodes.each_with_index do |node, idx|
        # needed for etcd client in edge
        dis.vnode_execute(node,
            "export ENDPOINTS=#{endpoints_str};rm -rf /root/etcdlog; mkdir /root/etcdlog") 
        addr = serv_node_ips[idx]
        # puts dis.vnode_execute(node, "etcd --version")
        cmd =  "nohup /usr/local/bin/etcd --name #{node} --initial-advertise-peer-urls http://#{addr}:2380 \
        --listen-peer-urls http://#{addr}:2380 \
        --listen-client-urls http://#{addr}:2379,http://127.0.0.1:2379 \
        --advertise-client-urls http://#{addr}:2379 \
        --initial-cluster-token #{cluster_token} \
        --initial-cluster #{initial_cluster_str} \
        --initial-cluster-state new > /root/etcdlog/etcd.log 2>&1 &"
        # --initial-cluster-state new &> /root/etcdlog/out.log &"
        # IMPORTANT: without the last part of the command the function blocks forever!
        # should we add this to listen-client-urls? ,http://127.0.0.1:4001
        puts dis.vnode_execute(node, cmd)
        # puts "etcd server #{idx+1} running"
    end
    sleep(7)
    dis.vnode_execute(server_vnodes[0], "etcdctl put mykey myvalue")
    sleep(3)
    out = dis.vnode_execute(server_vnodes[1], "etcdctl get mykey")
    if out.length>=2 && out[1] == "myvalue"
        puts "etcd cluster is working correctly"
    else
        puts "etcd cluster not setup correctly: '#{out}'"
    end
    dis.vnode_execute(server_vnodes[2], "etcdctl del mykey")
end
# puts "all etcd servers are now running!"