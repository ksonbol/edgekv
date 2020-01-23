#!/usr/bin/ruby -w

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
subnet = g5k.get_subnets(job).first
subnet_addr = "#{subnet.address}/#{subnet.prefix}"

pnodes = job['assigned_nodes']
pnodes.map!{|n| n.split(".")[0]}  # remove the ".SITE_NAME.grid5000.fr" suffix
# raise 'This experiment requires at least two physical machines' unless pnodes.size >= 2
coordinator = pnodes.first

# This ruby hash table describes our virtual network
vnet = {
  'name' => 'edgekvnet',
  'address' => subnet_addr
}

# running on coordinator node
private_key = IO.readlines('/root/.ssh/id_rsa').join
public_key = IO.readlines('/root/.ssh/id_rsa.pub').join

sshkeys = {
  'private' => private_key,
  'public' => public_key
}

# Connect to the Distem server (on http://localhost:4567 by default)
Distem.client do |dis|
  puts 'Creating virtual network'
  # Start by creating the virtual network
  dis.vnetwork_create(vnet['name'], vnet['address'])
  puts 'Creating virtual nodes'
  
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

  # # Creating one virtual node per physical one
  pnode_idx = 0
  VNODE_LIST.each_with_index do |node_name, idx|
    dis.vnode_create(node_name, { 'host' => pnodes[pnode_idx] }, sshkeys)
    # dis.vnode_create(node_name, { 'host' => pnodes[pnode_idx], 'vfilesystem' => \
    #   {'image' => FSIMG, 'path' => '/mnt/edgekv-1/etcd-1'}}, sshkeys)
    dis.vfilesystem_create(node_name, { 'image' => FSIMG })
    dis.viface_create(node_name, 'if0', { 'vnetwork' => vnet['name'] })
    pnode_idx += 1
  end

  puts 'Starting virtual nodes'
  sleep(2)
  # Starting the virtual nodes using the synchronous method
  VNODE_LIST.each do |nodename|
    dis.vnode_start(nodename)
  end

  sleep(5)
  if dis.wait_vnodes({'timeout' => 60}) # optional opts arg: {'timeout' => 600, 'port' => 22}, timeout in seconds
    puts "vnodes started successfully"
  else
    puts "vnodes are unreachable, maybe wait a little more?"
    exit 1
  end

  # allow internet access for all nodes
  VNODE_LIST.each_with_index do |node, idx|
    addr = dis.viface_info(node,'if0')['address'].split('/')[0]
    dis.vnode_execute(node, "ifconfig if0 #{addr} netmask 255.252.0.0;route add default gw 10.147.255.254 dev if0")
  end
end