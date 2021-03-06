#!/usr/bin/ruby

gem 'ruby-cute', ">=0.6"
require 'cute'
require 'pp' # pretty print
require 'distem'
require_relative 'conf'

SITE_NAME = "nancy"
HOSTNAME = "g5k"
# The path to the compressed filesystem image
# We can point to local file since our homedir is available from NFS
# FSIMG="file:///home/ksonbol/distem_img/distem-fs-jessie.tar.gz"
# FSIMG="file:///home/ksonbol/fs-img/edgekv-fs-jessie.tar.gz"
#TODO: should i use different images for client and server?
FSIMG="file:///home/ksonbol/fs-img/cli-1-fsimage.tar.gz"

g5k = Cute::G5K::API.new(:username => "ksonbol")
jobs = g5k.get_my_jobs(SITE_NAME)
raise "No jobs running! Run ruby platform_setup.rb --reserve to create a job" unless jobs.length() > 0

if jobs.length() > 1
  puts "WARNING: You have multiple jobs running at #{SITE_NAME}"
end

job = jobs.first
subnets = g5k.get_subnets(job)
subnet_addr = []
subnets.each do |subnet|
  subnet_addr << "#{subnet.address}/#{subnet.prefix}"
end

raise "Not enough subnets!" if subnets.length < NUM_VNETS

pnodes = job['assigned_nodes']
pnodes.map!{|n| n.split(".")[0]}  # remove the ".SITE_NAME.grid5000.fr" suffix
# raise 'This experiment requires at least two physical machines' unless pnodes.size >= 2

# running on coordinator node
private_key = IO.readlines('/root/.ssh/id_rsa').join
public_key = IO.readlines('/root/.ssh/id_rsa.pub').join

sshkeys = {
  'private' => private_key,
  'public' => public_key
}

puts 'Creating virtual networks'
# Connect to the Distem server (on http://localhost:4567 by default)
Distem.client do |dis|
  # Start by creating the virtual network
  sub_idx = 0
  (1..NUM_GROUPS).each do |i|
    dis.vnetwork_create("cli-#{i}", subnet_addr[sub_idx])
    dis.vnetwork_create("edge-#{i}", subnet_addr[sub_idx+1])
    dis.vnetwork_create("gw-#{i}", subnet_addr[sub_idx+2])
    sub_idx += 3
  end
  dis.vnetwork_create("gw-gw", subnet_addr[sub_idx])
end

puts "Vritual networks created"

# vnode_idx = 0
# for pnode in pnodes
  #   if vnode_idx >= NUM_VNODES
  #     break
  #   end
  #   for i in 1..2  # for each reserved disk in pnode
  #     if vnode_idx >= NUM_VNODES
  #       break
  #     end
  #     # assign a disk to each vnode
  #     dis.vnode_create(VNODE_LIST[vnode_idx],
  #       {'host' => pnode,
  #       'vfilesystem' => {
    #         'image' => FSIMG,
    #         'shared' => false,  # todo: check if we can use a shared file system
  #         'path' => "/mnt/edgekv-#{i}",
  #       }
  #       }, sshkeys)
  #     dis.viface_create(VNODE_LIST[vnode_idx], 'if0', { 'vnetwork' => vnet['name'] })
  #     vnode_idx += 1
  #   end
  # end
  
  # put vnode1 on pnode1 (coordinator), and the rest on pnode2
  
puts 'Creating virtual nodes'
# Creating one virtual node per physical one
pnode_idx = 0
group_idx = 1
Distem.client do |dis|
  SERVER_VNODES.each_with_index do |node_name, idx|
    dis.vnode_create(node_name, { 'host' => pnodes[pnode_idx] }, sshkeys)
    #   dis.vnode_create(node_name, { 'host' => pnodes[pnode_idx], 'vfilesystem' => \
    # {'image' => FSIMG, 'path' => '/mnt/edgekv-1/etcd-1'}}, sshkeys)
    dis.vfilesystem_create(node_name, { 'image' => FSIMG })
    dis.viface_create(node_name, 'if0', {'vnetwork' => "cli-#{group_idx}"})
    dis.viface_create(node_name, 'if1', {'vnetwork' => "edge-#{group_idx}"})
    dis.viface_create(node_name, 'if2', {'vnetwork' => "gw-#{group_idx}"})
    pnode_idx += 1
    if ((idx+1) % NUM_SRVR_PER_GROUP) == 0
      group_idx += 1
    end
  end
end

puts "Created edge nodes"

Distem.client do |dis|
  GATEWAY_VNODES.each_with_index do |node_name, idx|
    dis.vnode_create(node_name, { 'host' => pnodes[pnode_idx] }, sshkeys)
    dis.vfilesystem_create(node_name, { 'image' => FSIMG })
    dis.viface_create(node_name, 'if0', {'vnetwork' => "gw-#{idx+1}"}) # gw-edge
    dis.viface_create(node_name, 'if1', {'vnetwork' => "gw-gw"})
    pnode_idx += 1
  end
end

puts "Created gateway nodes"

Distem.client do |dis|
  # this assumes you have a single client per edge group
  CLIENT_VNODES.each_with_index do |node_name, idx|
    dis.vnode_create(node_name, { 'host' => pnodes[pnode_idx] }, sshkeys)
    dis.vfilesystem_create(node_name, { 'image' => FSIMG })
    dis.viface_create(node_name, 'if0', {'vnetwork' => "cli-#{idx+1}"})
    pnode_idx += 1
  end
end

puts "Created client nodes"

sleep(2)

puts 'Starting virtual nodes'
# Starting the virtual nodes using the synchronous method
Distem.client do |dis|
  VNODE_LIST.each do |nodename|
    dis.vnode_start(nodename)
  end
end

puts "All virtual nodes are running"
sleep(5)
  # if dis.wait_vnodes({'timeout' => 100}) # optional opts arg: {'timeout' => 600, 'port' => 22}, timeout in seconds
  #   puts "vnodes started successfully"
  # else
  #   puts "vnodes are unreachable, maybe wait a little more?"
  #   exit 1
  # end

# allow internet access for all nodes
puts "enabling internet access for all nodes"
Distem.client do |dis|
  VNODE_LIST.each_with_index do |node, idx|
    addr = dis.viface_info(node,'if0')['address'].split('/')[0] # todo: if0 should be enough here?
    dis.vnode_execute(node, "ifconfig if0 #{addr} netmask 255.252.0.0;route add default gw 10.147.255.254 dev if0")
  end
end

puts "internet access enabled for all nodes"
