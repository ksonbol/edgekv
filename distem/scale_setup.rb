
gem 'ruby-cute', ">=0.6"
require 'cute'
require 'pp' # pretty print
require 'distem'

SITE_NAME = "nancy"
HOSTNAME = "g5k"
# The path to the compressed filesystem image
# We can point to local file since our homedir is available from NFS
# FSIMG="file:///home/ksonbol/distem_img/distem-fs-jessie.tar.gz"
# FSIMG="file:///home/ksonbol/fs-img/edgekv-fs-jessie.tar.gz"
#TODO: should i use different images for client and server?
FSIMG="file:///home/ksonbol/fs-img/cli-1-fsimage.tar.gz"

num_gw = 1000
num_cli = 1
numNodes = num_gw + num_cli

g5k = Cute::G5K::API.new(:username => "ksonbol")
jobs = g5k.get_my_jobs(SITE_NAME)
raise "No jobs running! Run ruby platform_setup.rb --reserve to create a job" unless jobs.length() > 0

if jobs.length() > 1
  puts "WARNING: You have multiple jobs running at #{SITE_NAME}"
end

job = jobs.first
subnets = g5k.get_subnets(job)
subnet = subnets.first
subnet_addr = "#{subnet.address}/#{subnet.prefix}"

pnodes = job['assigned_nodes']
pnodes.map!{|n| n.split(".")[0]}  # remove the ".SITE_NAME.grid5000.fr" suffix

# running on coordinator node
private_key = IO.readlines('/root/.ssh/id_rsa').join
public_key = IO.readlines('/root/.ssh/id_rsa.pub').join

sshkeys = {
  'private' => private_key,
  'public' => public_key
}

vnet_name = "vnet"

puts 'Creating the virtual network'
# Connect to the Distem server (on http://localhost:4567 by default)
Distem.client do |dis|
  dis.vnetwork_create(vnet_name, subnet_addr)
end

puts "created virtual networks"

puts 'Creating virtual nodes'

gw_nodes = []
(1..num_gw).each do |i|
    gw_nodes << "gw-#{i}"
end

cli_nodes = []
(1..num_cli).each do |i|
    cli_nodes << "cli-#{i}"
end

nodes = gw_nodes + cli_nodes

Distem.client do |dis|
    gw_nodes.each do |node|
        dis.vnode_create(node, sshkeys)
        dis.vfilesystem_create(node, { 'image' => FSIMG, 'shared' => true })
        dis.viface_create(node, 'if0', {'vnetwork' => vnet_name})
    end

    cli_nodes.each do |node|
        dis.vnode_create(node, sshkeys)
        dis.vfilesystem_create(node, { 'image' => FSIMG, 'shared' => true })
        dis.viface_create(node, 'if0', {'vnetwork' => vnet_name})
    end
end

puts "created virtual nodes"

puts 'Starting virtual nodes'
# Starting the virtual nodes using the synchronous method
Distem.client do |dis|
  nodes.each do |nodename|
    dis.vnode_start(nodename)
  end
end

puts "All virtual nodes are running"
sleep(5)

puts "enabling internet access for all nodes"
Distem.client do |dis|
  nodes.each_with_index do |node, idx|
    addr = dis.viface_info(node,'if0')['address'].split('/')[0] # todo: if0 should be enough here?
    dis.vnode_execute(node, "ifconfig if0 #{addr} netmask 255.252.0.0;route add default gw 10.147.255.254 dev if0")
  end
end

puts "internet access enabled for all nodes"
