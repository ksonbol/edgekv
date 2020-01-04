#!/usr/bin/ruby
# Import the Distem module
require 'distem'
# The path to the compressed filesystem image
# We can point to local file since our homedir is available from NFS
FSIMG="file:///home/ksonbol/distem_img/distem-fs-jessie.tar.gz"
# Put the physical machines that have been assigned to you
# You can get that by executing: cat $OAR_NODE_FILE | uniq
pnodes=["grisou-3","grisou-4","grisou-5"]
raise 'This experiment requires at least two physical machines' unless pnodes.size >= 2
# The first argument of the script is the address (in CIDR format)
# of the virtual network to set-up in our platform
# This ruby hash table describes our virtual network
vnet = {
  'name' => 'testnet',
  'address' => ARGV[0]
}
nodelist = ['node-1','node-2']
# Read SSH keys
private_key = IO.readlines('/root/.ssh/id_rsa').join
public_key = IO.readlines('/root/.ssh/id_rsa.pub').join
sshkeys = {
  'private' => private_key,
  'public' => public_key
}
# Connect to the Distem server (on http://localhost:4567 by default)
Distem.client do |cl|
  puts 'Creating virtual network'
  # Start by creating the virtual network
  cl.vnetwork_create(vnet['name'], vnet['address'])
  # Creating one virtual node per physical one
  puts 'Creating virtual nodes'
  # Create the first virtual node and set it to be hosted on
  # the first physical machine
  cl.vnode_create(nodelist[0], { 'host' => pnodes[0] }, sshkeys)
  # Specify the path to the compressed filesystem image
  # of this virtual node
  cl.vfilesystem_create(nodelist[0], { 'image' => FSIMG })
  # Create a virtual network interface and connect it to vnet
  cl.viface_create(nodelist[0], 'if0', { 'vnetwork' => vnet['name'], 'default' => 'true' })
  # Create the first virtual node and set it to be hosted on
  # the second physical machine
  cl.vnode_create(nodelist[1], { 'host' => pnodes[1] }, sshkeys)
  cl.vfilesystem_create(nodelist[1], { 'image' => FSIMG })
  cl.viface_create(nodelist[1], 'if0', { 'vnetwork' => vnet['name'] })
  puts 'Starting virtual nodes'
  # Starting the virtual nodes using the synchronous method
  nodelist.each do |nodename|
    cl.vnode_start(nodename)
  end
end
