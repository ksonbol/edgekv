#!/usr/bin/ruby -w

gem 'ruby-cute', ">=0.6"
require 'cute'
require 'pp' # pretty print
require 'distem'

SITE_NAME = "nancy"
HOSTNAME = "g5k"
# The path to the compressed filesystem image
# We can point to local file since our homedir is available from NFS
# FSIMG="file:///home/ksonbol/distem_img/distem-fs-jessie.tar.gz"
FSIMG="file:///home/ksonbol/fs-img/edgekv-fs-jessie.tar.gz"

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
  'name' => 'testnet',
  'address' => subnet_addr
}

# server_vnodes = ['etcd-1', 'etcd-2', 'etcd-3', 'etcd-4', 'etcd-5']
server_vnodes = ['etcd-1', 'etcd-2', 'etcd-3']
client_vnodes = ['cli-1']
vnodelist = server_vnodes + client_vnodes
NUM_VNODE = vnodelist.length

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
  vnode_idx = 0
  for pnode in pnodes
    if vnode_idx >= vnodelist.length
      break
    end
    for i in 1..2  # for each reserved disk in pnode
      if vnode_idx >= vnodelist.length
        break
      end
      # assign a disk to each vnode
      dis.vnode_create(vnodelist[vnode_idx],
        {'host' => pnode,
        'vfilesystem' => {
          'image' => FSIMG,
          'shared' => false,  # todo: check if we can use a shared file system
          'path' => "/mnt/edgekv-#{i}",
        }
        }, sshkeys)
      dis.viface_create(vnodelist[vnode_idx], 'if0', { 'vnetwork' => vnet['name'] })
      vnode_idx += 1
    end
  end

  # # Creating one virtual node per physical one
  # # put vnode1 on pnode1 (coordinator), and the rest on pnode2
  # vnodelist.each_with_index do |node_name, idx|
  #   pnode_idx = idx==0 ? 0 : 1
  #   # dis.vnode_create(node_name, { 'host' => pnodes[pnode_idx] }, sshkeys)
  #   dis.vnode_create(node_name, { 'host' => pnodes[pnode_idx], 'vfilesystem' => \
  #     {'image' => FSIMG, 'path' => '/mnt/edgekv-1/etcd-1'}}, sshkeys)
  #   dis.vfilesystem_create(node_name, { 'image' => FSIMG })
  #   dis.viface_create(node_name, 'if0', { 'vnetwork' => vnet['name'] })
  #   dis.viface_create('etcd-1', 'if0', { 'vnetwork' => vnet['name'] })
  # end

  puts 'Starting virtual nodes'
  # Starting the virtual nodes using the synchronous method
  vnodelist.each do |nodename|
    dis.vnode_start(nodename)
  end

  sleep(5)
  if dis.wait_vnodes({'timeout' => 60}) # optional opts arg: {'timeout' => 600, 'port' => 22}, timeout in seconds
    puts "vnodes started successfully"
  else
    puts "vnodes are unreachable, maybe wait a little more?"
    exit 1
  end

  # puts "Updating vnodes latencies"
  cc = 0.035  # cloud-to-cloud latency
  cl = 60    # cloud-to-client latency
  ee = 3.2    # edge-to-edge latency
  el = 3.2    # edge-to-client latency
  
  latency_mat = Array.new(NUM_VNODE) {Array.new(NUM_VNODE, 0)}
  # assume lat(node to itself) = 0
  # assume lat(client to client) = 0

  # # set edge latencies (in ms??)
  # for r in 0..NUM_VNODE-1
  #   for c in 0..NUM_VNODE-1
  #     if r != c  # we let node-to-self latency = 0
  #       if (r < server_vnodes.length) && (c < server_vnodes.length)
  #         # edge-to-edge
  #         latency_mat[r][c] = ee
  #       elsif (r >= server_vnodes.length) ^ (c >= server_vnodes.length)  # xor
  #         # edge-to-client
  #         latency_mat[r][c] = el
  #       # else: client-to-client already zeroed
  #       end
  #     end
  #   end
  # end

  # dis.set_peers_latencies(vnodelist, latency_mat)
  # put "updated latencies for edge experiment"
  # start edge experiment


  # set cloud latencies (in ms??)
  latency_mat = Array.new(NUM_VNODE) {Array.new(NUM_VNODE, 0)}
  for r in 0..NUM_VNODE-1
    for c in 0..NUM_VNODE-1
      if r != c  # we let node-to-self latency = 0
        if (r < server_vnodes.length) && (c < server_vnodes.length)
          # cloud-to-cloud
          latency_mat[r][c] = cc
        elsif (r >= server_vnodes.length) ^ (c >= server_vnodes.length)  # xor
          # cloud-to-client
          latency_mat[r][c] = cl
        # else: client-to-client already zeroed
        end
      end
    end
  end

  dis.set_peers_latencies(vnodelist, latency_mat)

  # start cloud experiment



    # # set edge latencies (in ms??)
  # edge_lat = Matrix[
  #   #  s1  s2  s3  s4  s5  cl1  
  #     [0,  ee, ee, ee, ee, el]   #s1
  #     [ee, 0,  ee, ee, ee, el]   #s2
  #     [ee, ee, 0,  ee, ee, el]   #s3
  #     [ee, ee, ee,  0, ee, el]   #s4
  #     [ee, ee, ee, ee,  0, el]   #s5  
  #     [el, el, el, el, el, 0 ]   #cl1  
  # ]
  # dis.set_peers_latencies(vnodelist, edge_lat)


#   cloud_lat = Matrix[
#   #  s1  s2  s3  s4  s5  cl1  
#     [0,  cc, cc, cc, cc, cl]   #s1
#     [cc, 0,  cc, cc, cc, cl]   #s2
#     [cc, cc, 0,  cc, cc, cl]   #s3
#     [cc, cc, cc,  0, cc, cl]   #s4
#     [cc, cc, cc, cc,  0, cl]   #s5  
#     [cl, cl, cl, cl, cl, 0 ]   #cl1  
# ]
  # dis.set_peers_latencies(vnodelist, cloud_lat)

  # start cloud experiment

end

# After finishing the experiment, cancel the job (reservation) if needed
# g5k.release(job)