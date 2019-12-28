#!/usr/bin/ruby -w

gem 'ruby-cute', ">=0.6"
require 'cute'
require 'pp' # pretty print
require 'distem'

SITE_NAME = "nancy"
HOSTNAME = "g5k"
# The path to the compressed filesystem image
# We can point to local file since our homedir is available from NFS
FSIMG="file:///home/ksonbol/distem_img/distem-fs-jessie.tar.gz"

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

# This ruby hash table describes our virtual network
vnet = {
  'name' => 'testnet',
  'address' => subnet_addr
}

nodelist = ['cli-1', 'edge-1', 'edge-2', 'edge-3']

########
# if running remotely

# connect to coordinator
# Net::SSH.start("#{coordinator}", "root") do |ssh|
#   # Read SSH keys
  # private_key = ssh.exec!("cat ~/.ssh/id_rsa")
  # public_key = ssh.exec!("cat ~/.ssh/id_rsa.pub")
# end
# Net::SSH::Multi.start do |ssh|
  # ssh.use "root@#{pnodes.first}"

# Net::SSH.start("#{SITE_NAME}.#{HOSTNAME}") do |ssh|
# end
#########
# if running on coordinator node
private_key = IO.readlines('/root/.ssh/id_rsa').join
public_key = IO.readlines('/root/.ssh/id_rsa.pub').join

# if running on frontend (note that frontend keys and coordinator keys are the same, we copied them)
# private_key = IO.readlines('/home/ksonbol/.ssh/id_rsa').join
# public_key = IO.readlines('/home/ksonbol/.ssh/id_rsa.pub').join
#########

sshkeys = {
  'private' => private_key,
  'public' => public_key
}

puts sshkeys

# Connect to the Distem server (on http://localhost:4567 by default)
Distem.client do |cl|
  puts 'Creating virtual network'
  # Start by creating the virtual network
  cl.vnetwork_create(vnet['name'], vnet['address'])
  # Creating one virtual node per physical one
  puts 'Creating virtual nodes'
  # put vnode1 on pnode1 (coordinator), and the rest on pnode2
  nodelist.each_with_index do |node_name, idx|
    pnode_idx = idx==0 ? 0 : 1
    cl.vnode_create(node_name, { 'host' => pnodes[pnode_idx] }, sshkeys)
    cl.vfilesystem_create(node_name, { 'image' => FSIMG })
    cl.viface_create(node_name, 'if0', { 'vnetwork' => vnet['name'] })
  end
  puts 'Starting virtual nodes'
  # Starting the virtual nodes using the synchronous method
  nodelist.each do |nodename|
    cl.vnode_start(nodename)
  end
end

# After finishing the experiment, cancel the job (reservation) if needed
# g5k.release(job)