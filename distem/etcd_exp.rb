#!/usr/bin/ruby

gem 'ruby-cute', ">=0.6"
require 'cute'
require 'pp' # pretty print
require 'optparse'
require_relative 'conf'

NUM_NODES = 4
HOSTNAME = 'g5k'  # that is the alias I used, the second alias allows accessing the site directly
SITE_NAME = 'nancy'
CLUSTER_NAME = 'grimoire'
WALL_TIME = '03:00:00'
JOB_QUEUE = 'default'  # use 'default' or 'besteffort'

g5k = Cute::G5K::API.new()

job = g5k.reserve(:site => SITE_NAME, :type => :deploy, :queue => JOB_QUEUE,
    :resources => "/slash_22=1+{(type='disk' or type='default') and cluster='#{CLUSTER_NAME}'} \
                   /nodes=#{NUM_NODES}",
                    :keys => SSH_KEY_PATH,
                    :walltime => WALL_TIME)

# job = g5k.reserve(:site => 'nancy', :type => :deploy, :queue => 'default',
#     :resources => "/slash_22=1+{(type='disk' or type='default') and cluster='grimoire'} \
#                    /nodes=4,walltime=02:30:00",
#                    :keys => '/home/ksonbol/.ssh/id_rsa')

                   # :reservation => RESERV_TIME,  # for future reservations
puts "Assigned nodes : #{job['assigned_nodes']}"
g5k.deploy(job, :env => "debian9-x64-nfs", :keys => SSH_KEY_PATH, :wait => true)
nodes = job['assigned_nodes']
coordinator = nodes.first

puts "Preparing node file"
NODEFILES_DIR = "/home/ksonbol/jobs"
NODEFILE = File.join(NODEFILES_DIR, job['uid'].to_s)
open(NODEFILE, 'w') { |f|
  nodes.each do |node|
      f.puts "#{node}\n"
  end
}

system("bash -c 'distem-bootstrap --enable-admin-network -f #{NODEFILE} -k #{SSH_KEY_PATH} --debian-version stretch'")

puts "installing ruby-cute gem on coordinator node for g5k communication"
# since this is stored in our fs image, we probably dont need to run this any more
system("ssh root@#{coordinator} 'gem install ruby-cute'")
if $?.exitstatus == 0
  puts "gem installed successfully"
end

######### RUN THIS ON THE COORDINATOR NODE ###########
require 'distem'
gem 'ruby-cute', ">=0.6"
require 'cute'
SITE_NAME = "nancy"
FSIMG="file:///home/ksonbol/fs-img/scale-fsimage.tar.gz"

g5k = Cute::G5K::API.new(:username => "ksonbol")
job = g5k.get_my_jobs(SITE_NAME).first
subnet = g5k.get_subnets(job).first
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

num_server = 3
num_cli = 1
vnet_name = "vnet"
puts 'Creating virtual networks'
# Connect to the Distem server (on http://localhost:4567 by default)
Distem.client do |dis|
  # Start by creating the virtual network
  dis.vnetwork_create(vnet_name, subnet_addr)
end

puts 'Creating virtual nodes'

srv_nodes = []
(1..num_server).each do |i|
    srv_nodes << "etcd-#{i}"
end

cli_nodes = []
(1..num_cli).each do |i|
    cli_nodes << "cli-#{i}"
end

nodes = srv_nodes + cli_nodes
pnode_idx = 0

Distem.client do |dis|
    srv_nodes.each do |node|
        dis.vnode_create(node, { 'host' => pnodes[pnode_idx] }, sshkeys)
        dis.vfilesystem_create(node, { 'image' => FSIMG })
        dis.viface_create(node, 'if0', {'vnetwork' => vnet_name})
        pnode_idx += 1
    end

    cli_nodes.each do |node|
        dis.vnode_create(node, { 'host' => pnodes[pnode_idx] }, sshkeys)
        dis.vfilesystem_create(node, { 'image' => FSIMG})
        dis.viface_create(node, 'if0', {'vnetwork' => vnet_name})
        pnode_idx += 1
    end
end

puts 'Starting virtual nodes'
# Starting the virtual nodes using the synchronous method
Distem.client do |dis|
  nodes.each do |nodename|
    dis.vnode_start(nodename)
  end
end

sleep(5)

puts "enabling internet access for all nodes"
Distem.client do |dis|
  nodes.each_with_index do |node, idx|
    addr = dis.viface_info(node,'if0')['address'].split('/')[0] # todo: if0 should be enough here?
    dis.vnode_execute(node, "ifconfig if0 #{addr} netmask 255.252.0.0;route add default gw 10.147.255.254 dev if0")
  end
end

puts "internet access enabled for all nodes"

####### RUN THESE COMMANDS ON ETCD SERVER NODES #######

PROMETHEUS_VERSION="2.0.0"
wget https://github.com/prometheus/prometheus/releases/download/v$PROMETHEUS_VERSION/prometheus-$PROMETHEUS_VERSION.linux-amd64.tar.gz -O /tmp/prometheus-$PROMETHEUS_VERSION.linux-amd64.tar.gz
tar -xvzf /tmp/prometheus-$PROMETHEUS_VERSION.linux-amd64.tar.gz --directory /tmp/ --strip-components=1
/tmp/prometheus --version

cat > /tmp/test-etcd.yaml <<EOF
global:
  scrape_interval: 10s
scrape_configs:
  - job_name: test-etcd
    static_configs:
    - targets: ['10.144.0.1:2379','10.144.0.2:2379','110.144.0.3:2379']
EOF
cat /tmp/test-etcd.yaml


pkill etcd
rm -rf /root/etcdlog /root/*.etcd; mkdir /root/etcdlog
nohup /usr/local/bin/etcd --heartbeat-interval=10 --election-timeout=100 \
            --name etcd-1 --initial-advertise-peer-urls http://10.144.0.1:2379 \
            --listen-peer-urls http://10.144.0.1:2379 \
            --listen-client-urls http://10.144.0.1:2380,http://127.0.0.1:2380 \
            --advertise-client-urls http://10.144.0.1:2380 \
            --initial-cluster-token cluster \
            --initial-cluster etcd-1=http://10.144.0.1:2379,etcd-2=http://10.144.0.2:2379,etcd-3=http://10.144.0.3:2379 \
            --initial-cluster-state new > /root/etcdlog/etcd.log 2>&1 &

cd /root
pkill etcd
rm -rf /root/etcdlog /root/*.etcd; mkdir /root/etcdlog
nohup /usr/local/bin/etcd --heartbeat-interval=10 --election-timeout=100 \
            --name etcd-2 --initial-advertise-peer-urls http://10.144.0.2:2379 \
            --listen-peer-urls http://10.144.0.2:2379 \
            --listen-client-urls http://10.144.0.2:2380,http://127.0.0.1:2380 \
            --advertise-client-urls http://10.144.0.2:2380 \
            --initial-cluster-token cluster \
            --initial-cluster etcd-1=http://10.144.0.1:2379,etcd-2=http://10.144.0.2:2379,etcd-3=http://10.144.0.3:2379 \
            --initial-cluster-state new > /root/etcdlog/etcd.log 2>&1 &

cd /root
pkill etcd
rm -rf /root/etcdlog /root/*.etcd; mkdir /root/etcdlog
nohup /usr/local/bin/etcd --heartbeat-interval=10 --election-timeout=100 \
            --name etcd-3 --initial-advertise-peer-urls http://10.144.0.3:2379 \
            --listen-peer-urls http://10.144.0.3:2379 \
            --listen-client-urls http://10.144.0.3:2380,http://127.0.0.1:2380 \
            --advertise-client-urls http://10.144.0.3:2380 \
            --initial-cluster-token cluster \
            --initial-cluster etcd-1=http://10.144.0.1:2379,etcd-2=http://10.144.0.2:2379,etcd-3=http://10.144.0.3:2379 \
            --initial-cluster-state new > /root/etcdlog/etcd.log 2>&1 &

## RUN THIS AFTER STARTING THE ETCD NODES
nohup /tmp/prometheus \
    --config.file /tmp/test-etcd.yaml \
    --web.listen-address ":9090" \
    --storage.tsdb.path "test-etcd.data" >> /tmp/test-etcd.log  2>&1 &

####### RUN THESE COMMANDS ON THE CLIENT NODE #######
export GOPATH=$HOME/go
go get go.etcd.io/etcd/tools/benchmark
export HOST_1=10.144.0.1:2379
export HOST_2=10.144.0.2:2379
export HOST_3=10.144.0.3:2379
$GOPATH/bin/benchmark --endpoints 10.144.0.2:2379,10.144.0.3:2379 --clients 1 put


### SETTING LATENCY, RUN FROM COORDINATOR ###
require 'distem'
num_server = 3
num_cli = 1
vnet_name = "vnet"

srv_nodes = []
(1..num_server).each do |i|
    srv_nodes << "etcd-#{i}"
end

cli_nodes = []
(1..num_cli).each do |i|
    cli_nodes << "cli-#{i}"
end

nodes = srv_nodes + cli_nodes

# edge latencies

Distem.client do |dis|
    # etcd-etcd delay = 2ms
    srv_nodes.each do |node|
        dis.viface_update(node, 'if0',{
            "output" => {"latency"=>{"delay"=> "1ms"}},
            "input" => {"latency"=>{"delay"=> "1ms"}}
          }
        )
    end

    # etcd-cli delay = 5ms
    cli_nodes.each do |node|
        dis.viface_update(node, 'if0',{
            "output" => {"latency"=>{"delay"=> "4ms"}},
            "input" => {"latency"=>{"delay"=> "4ms"}}
          }
        )
    end
end

# cloud latencies
Distem.client do |dis|
    # etcd-etcd delay = 0.05ms (rtt becomes around 0.12)
    srv_nodes.each do |node|
        dis.viface_update(node, 'if0',{
            "output" => {"latency"=>{"delay"=> "0.025ms"}},
            "input" => {"latency"=>{"delay"=> "0.025ms"}}
          }
        )
    end

    # etcd-cli delay = 50ms
    cli_nodes.each do |node|
        dis.viface_update(node, 'if0',{
            "output" => {"latency"=>{"delay"=> "50ms"}},
            "input" => {"latency"=>{"delay"=> "50ms"}}
          }
        )
    end
end

### Mount the SSD drive and run etcd servers over it!
mkdir /mnt/etcd

# if partition is not created:
echo 'type=83' | sfdisk /dev/sdf

mkfs.ext4 -F "/dev/sdf1"

mount "/dev/sdf1" /mnt/etcd

