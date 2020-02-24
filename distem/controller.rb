#!/usr/bin/ruby

gem 'ruby-cute', ">=0.6"
require 'cute'
require 'pp' # pretty print
require 'optparse'
require_relative 'conf'

options = {}

OptionParser.new do |opts|
  opts.banner = "Usage: ruby controller.rb [options]"

  opts.on("-s", "--setup", "perform platform setup") do |v|
    options[:setup] = v
  end

  opts.on("-r", "--reserve", "make node reservations in setup") do |v|
    options[:reserve] = v
  end

  opts.on("-d", "--deploy", "deploy filesystem to pnodes in setup") do |v|
    options[:deploy] = v
  end

  opts.on("-p", "--play", "start and test etcd and edge servers") do |v|
    options[:play] = v
  end

  opts.on("-l", "--set-latency ENVIRONMENT", "Set latencies for 'cloud' or 'edge' environments") do |v|
    options[:latency] = v
  end

  opts.on("-e", "--experiment NUMBER", "Run experiment with given number") do |v|
    options[:exp] = v
  end
end.parse!

NUM_NODES = NUM_VNODES
HOSTNAME = 'g5k'  # that is the alias I used, the second alias allows accessing the site directly
SITE_NAME = 'nancy'
CLUSTER_NAME = 'grisou'
WALL_TIME = '05:00:00'
JOB_QUEUE = 'default'  # use 'default' or 'besteffort'
NODEFILES_DIR = "/home/ksonbol/jobs"
SETUP_FILE = 'setup.rb'
SRVR_EXP_FILE = 'server_exp.rb' # etcd servers
EDGE_EXP_FILE = 'edge_exp.rb'
CLI_EXP_FILE = 'client_exp.rb'
GW_EXP_FILE = 'gateway_exp.rb'
YCSB_COPY_FILE = 'ycsb_copy.rb'
YCSB_LOAD_FILE = 'ycsb_load.rb'
YCSB_RUN_FILE = 'ycsb_run.rb'
YCSB_EXP_FILE = 'ycsb_exp.rb'
SET_LAT_FILE = 'set_lat.rb'
CONF_FILE = 'conf.rb'
EDGEKV_FOLDER = "edgekv"
DISK_PREP_SH = "disk_prep.sh"
RESERVE_AFTER = 22 # minutes

t = Time.now.round()
t += RESERVE_AFTER * 60
RESERV_TIME = t.strftime("%Y-%m-%d %H:%M:%S")

# Grid'5000  part
g5k = Cute::G5K::API.new()

# When the script is run with --reserve, node reservation will be done
# if ARGV[0] == '--reserve'
if options[:setup]
  if options[:reserve]
    # reserve and deploy the nodes
    # job = g5k.reserve(:nodes => NUM_NODES, :site => SITE_NAME, :walltime => WALL_TIME,
    #                   :env => "debian9-x64-nfs", :subnets => [22,1], :keys => SSH_KEY_PATH)

    # only reserve the nodes
    job = g5k.reserve(:nodes => NUM_NODES, :site => SITE_NAME,
                      :cluster => CLUSTER_NAME,
                      :walltime => WALL_TIME, :subnets => [22, NUM_VNETS], :type => :deploy,
                      # :reservation => RESERV_TIME,  # for future reservations
                      :queue => JOB_QUEUE)
                      # :keys => SSH_KEY_PATH)
    
    # reserve nodes using resource (oar) options
    # job = g5k.reserve(:site => SITE_NAME, :resources => "slash_22=1+nodes=#{NUM_NODES},walltime=#{WALL_TIME}",
    #                   :type => :deploy, :queue => JOB_QUEUE)

    # job = g5k.reserve(:site => SITE_NAME, :type => :deploy, :queue => JOB_QUEUE,
    #                   :resources => "/slash_22=1+{(type='disk' or type='default') and cluster='#{CLUSTER_NAME}'} \
    #                                  /nodes=#{NUM_NODES},walltime=#{WALL_TIME}",
    #                   # :reservation => RESERV_TIME,  # for future reservations
                      # :keys => SSH_KEY_PATH)
    puts "Assigned nodes : #{job['assigned_nodes']}"
  else
    # otherwise, get the running job
    jobs = g5k.get_my_jobs(SITE_NAME)
    raise "No jobs running! Run script with --reserve to create a job" unless jobs.length() > 0
    job = jobs.first
  end

  if options[:deploy]
    g5k.deploy(job, :env => "debian9-x64-nfs", :keys => SSH_KEY_PATH, :wait => true)

    # preparing the disks
    # nodes =job['assigned_nodes']
    # nodes.each_with_index do |node, idx|
      # puts "preparing disks on node #{node}"
      # system("scp #{DISK_PREP_SH} root@#{node}:/root")
      # system("ssh root@#{node} 'mkdir /mnt/edgekv-1 /mnt/edgekv-2 /mnt/edgekv-3 /mnt/edgekv-4'")
      # system("ssh root@#{node} 'chmod a+x #{DISK_PREP_SH}'")
      # system("ssh root@#{node} 'sudo ./#{DISK_PREP_SH} /dev/sdb /mnt/edgekv-1'")
      # system("ssh root@#{node} 'sudo ./#{DISK_PREP_SH} /dev/sdc /mnt/edgekv-2'")
      # system("ssh root@#{node} 'sudo ./#{DISK_PREP_SH} /dev/sdd /mnt/edgekv-3'")
      # system("ssh root@#{node} 'sudo ./#{DISK_PREP_SH} /dev/sde /mnt/edgekv-4'")
    # end
  end

  
  nodes = job['assigned_nodes']
  coordinator = nodes.first
  
  puts "Preparing node file"
  NODEFILE = File.join(NODEFILES_DIR, job['uid'].to_s)
  open(NODEFILE, 'w') { |f|
    nodes.each do |node|
        f.puts "#{node}\n"
    end
  }

  # puts "Setting up Distem"
  # two ways to use my modified version for editing path location
  # system("bash -c 'distem-bootstrap -g --git-url https://github.com/ksonbol/distem.git -f #{NODEFILE} -k #{SSH_KEY_PATH} --debian-version stretch'")
  # system("bash -c '/home/ksonbol/distem-bootstrap-karim -g -f #{NODEFILE} -k #{SSH_KEY_PATH} --debian-version stretch'")
  
  system("bash -c 'distem-bootstrap --enable-admin-network -f #{NODEFILE} -k #{SSH_KEY_PATH} --debian-version stretch'")

  puts "installing ruby-cute gem on coordinator node for g5k communication"
  # since this is stored in our fs image, we probably dont need to run this any more
 system("ssh root@#{coordinator} 'gem install ruby-cute'")
  if $?.exitstatus == 0
    puts "gem installed successfully"
  end

  puts "Copying the setup file"
  system("scp #{SETUP_FILE} root@#{coordinator}:/root")
  system("scp #{CONF_FILE} root@#{coordinator}:/root")
  if $?.exitstatus == 0
    puts "copied setup file"
  end

  # puts "Setting up the virtual network"
  # system("ssh root@#{coordinator} 'ruby #{SETUP_FILE}'")
  # if $?.exitstatus == 0
  #   puts "virtual network set up successfully"
  # end

else  # skipping setup step
  jobs = g5k.get_my_jobs(SITE_NAME)
  raise "No jobs running! Run script with --reserve to create a job" unless jobs.length() > 0
  job = jobs.first
  nodes = job['assigned_nodes']
  coordinator = nodes.first
end

if options[:play]
  puts "copying experiment files to coordinator node"
  %x(scp #{CONF_FILE} root@#{coordinator}:/root)
  %x(scp #{SRVR_EXP_FILE} root@#{coordinator}:/root)
  %x(scp #{EDGE_EXP_FILE} root@#{coordinator}:/root)
  %x(scp #{CLI_EXP_FILE} root@#{coordinator}:/root)
  %x(scp #{GW_EXP_FILE} root@#{coordinator}:/root)
  %x(scp #{YCSB_COPY_FILE} root@#{coordinator}:/root)
  %x(scp #{YCSB_LOAD_FILE} root@#{coordinator}:/root)
  %x(scp #{YCSB_RUN_FILE} root@#{coordinator}:/root)
  %x(scp #{SET_LAT_FILE} root@#{coordinator}:/root)
  %x(scp -r #{EDGEKV_FOLDER} root@#{coordinator}:/root)
  %x(scp play.sh root@#{coordinator}:/root)
  %x(scp -r go-ycsb/ root@#{coordinator}:/root)
  %x(scp edgekv.conf root@#{coordinator}:/root)
  %x(scp cleanup.rb root@#{coordinator}:/root)
  %x(ssh #{coordinator} 'chmod a+x *.rb *.sh')

  # out = %x(scp -r edge/ root@#{coordinator}:/root)
  # out = %x(scp -r client/ root@#{coordinator}:/root)
  if $?.exitstatus == 0
    sleep(2)  # seconds
    puts "copying completed successfully"
  end

  # puts "Running experiments"

  # puts "Starting etcd servers"
  # system("ssh #{coordinator} 'ruby #{SRVR_EXP_FILE}'")
  # if $?.exitstatus == 0
  #   puts "etcd servers are running"
  # end

  # puts "Starting edge servers"
  # system("ssh #{coordinator} 'ruby #{EDGE_EXP_FILE}'")
  # if $?.exitstatus == 0
  #   puts "edge servers are running"
  # end
    
  # puts "Starting gateway nodes"
  # system("ssh #{coordinator} 'ruby #{GW_EXP_FILE}'")
  # if $?.exitstatus == 0
  #   puts "gateway nodes are running"
  # end

  # puts "Starting clients"
  # system("ssh #{coordinator} 'ruby #{CLI_EXP_FILE}'")
  # if $?.exitstatus == 0
  #   puts "clients are running"
  # end

  # puts "Copying and building YCSB files"
  # system("ssh #{coordinator} 'ruby #{YCSB_COPY_FILE}'")
  # if $?.exitstatus == 0
  #   puts "YCSB files copied and built"
  # end

end

# def ssh_run(remote, cmd, desc="command")
#   out = system("ssh #{remote} '#{cmd}'")
# end

# def scp(files, dest, is_dir=false)
#   rec_flag = is_dir ? "-r " : ""
#   success = system("scp #{rec_flag}#{files} #{dest}")
# end

if options[:latency]
  env = options[:latency]
  puts "Setting latency for environment #{env}"
  system("ssh #{coordinator} 'ruby #{SET_LAT_FILE} #{env}'")
  if $?.exitstatus == 0
    puts "setting latency finished"
  end
end

if options[:exp]
  exp_num = options[:exp]
  puts "Running YCSB experiment #{exp_num}"
  system("ssh #{coordinator} 'ruby #{YCSB_EXP_FILE} #{exp_num}'")
  if $?.exitstatus == 0
    puts "running experiments finished"
  end
end