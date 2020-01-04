#!/usr/bin/ruby -w

gem 'ruby-cute', ">=0.6"
require 'cute'
require 'pp' # pretty print
require 'optparse'
require 'time'

options = {}

OptionParser.new do |opts|
  opts.banner = "Usage: platform_setup.rb [options]"

  opts.on("-s", "--setup", "perform platform setup") do |v|
    options[:setup] = v
  end

  opts.on("-r", "--reserve", "make node reservations in setup") do |v|
    options[:reserve] = v
  end

  opts.on("-d", "--deploy", "deploy OS to pnodes") do |v|
    options[:deploy] = v
  end

  opts.on("-p", "--play", "run experiment") do |v|
    options[:play] = v
  end
end.parse!

NUM_NODES = 2
HOSTNAME = 'g5k'  # that is the alias I used, the second alias allows accessing the site directly
SITE_NAME = 'nancy'
CLUSTER_NAME = 'grisou'
# CLUSTER_NAME = 'gros'
WALL_TIME = '05:00:00'
SSH_KEY_PATH = '/home/ksonbol/.ssh/id_rsa'
JOB_QUEUE = 'default'  # use 'default' or 'besteffort'
NODEFILES_DIR = "/home/ksonbol/jobs"
SETUP_FILE = 'setup.rb'
SRVR_EXP_FILE = 'server_exp.rb' # etcd servers
EDGE_EXP_FILE = 'edge_exp.rb'
CLI_EXP_FILE = 'client_exp.rb'
GW_EXP_FILE = 'gateway_exp.rb'
EDGEKV_FOLDER = "edgekv"
RESERVE_AFTER = 2 # minutes

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
    job = g5k.reserve(:nodes => NUM_NODES, :site => SITE_NAME, :cluster => CLUSTER_NAME,
                      :walltime => WALL_TIME, :subnets => [22,1], :type => :deploy,
                      # :reservation => RESERV_TIME,  # for future reservations
                      :queue => JOB_QUEUE)
    
    # reserve nodes using resource (oar) options
    # job = g5k.reserve(:site => SITE_NAME, :resource => "slash_22=1+nodes=#{NUM_NODES},walltime=#{WALL_TIME}",
    #                   :type => :deploy, :queue => JOB_QUEUE)
    puts "Assigned nodes : #{job['assigned_nodes']}"
  else
    # otherwise, get the running job
    jobs = g5k.get_my_jobs(SITE_NAME)
    raise "No jobs running! Run script with --reserve to create a job" unless jobs.length() > 0
    job = jobs.first
  end

  if options[:deploy]
    g5k.deploy(job, :env => "debian9-x64-nfs", :keys => SSH_KEY_PATH, :wait => true)
  end

  puts "Preparing node file"
  nodes = job['assigned_nodes']
  NODEFILE = File.join(NODEFILES_DIR, job['uid'].to_s)
  open(NODEFILE, 'w') { |f|
    nodes.each do |node|
      f.puts "#{node}\n"
    end
  }

  puts "Setting up Distem"
  out = %x(bash -c "distem-bootstrap -f #{NODEFILE} -k #{SSH_KEY_PATH} --debian-version stretch")
  puts out
  if $?.exitstatus == 0  # exit status of last executed shell command
    puts "Distem setup completed successfully"
  end

  coordinator = nodes.first

  puts "installing ruby-cute gem on coordinator node for g5k communication"
  # since this is stored in our fs image, we probably dont need to run this any more
  out = %x(ssh root@#{coordinator} 'gem install ruby-cute')
  if $?.exitstatus == 0
    puts "gem installed successfully"
  end

  # set up the virtual network
  puts "Setting up the virtual network"
  out = %x(scp #{SETUP_FILE} root@#{coordinator}:/root)
  puts out
  if $?.exitstatus == 0
    puts "copied setup file"
  end
  
  out = %x(ssh root@#{coordinator} 'ruby #{SETUP_FILE}')
  puts out
  if $?.exitstatus == 0
    puts "virtual network set up successfully"

  end

else  # skipping setup step
  jobs = g5k.get_my_jobs(SITE_NAME)
  raise "No jobs running! Run script with --reserve to create a job" unless jobs.length() > 0
  job = jobs.first
  nodes = job['assigned_nodes']
  coordinator = nodes.first
end

if options[:play]
  puts "copying experiment files to coordinator node"
  out = %x(scp #{SRVR_EXP_FILE} root@#{coordinator}:/root)
  out = %x(scp #{EDGE_EXP_FILE} root@#{coordinator}:/root)
  out = %x(scp #{CLI_EXP_FILE} root@#{coordinator}:/root)
  out = %x(scp #{GW_EXP_FILE} root@#{coordinator}:/root)
  out = %x(scp -r #{EDGEKV_FOLDER} root@#{coordinator}:/root)
  # out = %x(scp -r edge/ root@#{coordinator}:/root)
  # out = %x(scp -r client/ root@#{coordinator}:/root)
  if $?.exitstatus == 0
    sleep(2)  # seconds
    puts "copying completed successfully"
  end

  puts "Running experiments"

  puts "Starting etcd servers"
  out = %x(ssh root@#{coordinator} 'ruby #{SRVR_EXP_FILE}')
  if $?.exitstatus == 0
    puts "etcd servers are running"
  end
  puts out

  puts "Starting edge servers"
  out = %x(ssh #{coordinator} 'ruby #{EDGE_EXP_FILE}')
  if $?.exitstatus == 0
    puts "edge servers are running"
  end
  puts out

  puts "Starting clients"
  out = %x(ssh #{coordinator} 'ruby #{CLI_EXP_FILE}')
  puts out
  if $?.exitstatus == 0
    puts "clients are running"
  end

  # puts "Starting gateway nodes"
  # out = %x(ssh root@#{coordinator} 'ruby #{GW_EXP_FILE}')
  # if $?.exitstatus == 0
  #   puts "gateway nodes are running"
  # end
end