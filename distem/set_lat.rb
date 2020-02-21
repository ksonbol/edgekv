#!/usr/bin/ruby -w

require 'distem'
require_relative 'conf'

no_threads = 100 # the number of threads to be used in each ycsb client
single_cl_bw = 100
total_cl_bw = no_threads * single_cl_bw

raise "Usage: ruby set_lat.rb ENVIRONMENT" unless ARGV.length == 1

  puts "Updating vnodes latencies and bandwidths"
  
  case ARGV[0]
  when "cloud"
    # unit is ms (millisecond)
    # these are interface latencies, link latency is double these values
    ee = "0.03ms"     # edge-to-edge server latency
    ee_bw = "1000mbps"
    el = "25ms"       # cloud-to-client latency
    el_bw = "#{total_cl_bw}mbps"
    eg = "0.03ms"     # server-to-gateway latency
    eg_bw = "1000mbps"
    gg = "0.03ms"     # gateway-to-gateway latency
    gg_bw = "1000mbps"
  when "edge"
    ee = "1ms"     # edge-to-edge latency
    ee_bw = "1000mbps"
    el = "2.5ms"     # edge-to-client latency
    el_bw = "#{total_cl_bw}mbps"
    eg = "1ms"     # edge-to-gateway latency
    eg_bw = "750mbps"
    gg = "5ms"    # gateway-to-gateway latency
    gg_bw = "500mbps"
  end


  Distem.client do |dis|
    SERVER_VNODES.each_with_index do |node_name, idx|
      dis.viface_update(node_name, 'if0', { # cli
        "output" => {"bandwidth"=>{"rate"=> el_bw}, "latency"=>{"delay"=> el }},
        "input" => {"bandwidth"=>{"rate"=> el_bw}, "latency"=>{"delay"=> el }}
      })
      dis.viface_update(node_name, 'if1', { # edge
        "output" => {"bandwidth"=>{"rate"=> ee_bw}, "latency"=>{"delay"=> ee }},
        "input" => {"bandwidth"=>{"rate"=> ee_bw}, "latency"=>{"delay"=> ee }}
      })
      dis.viface_update(node_name, 'if2', { # gw
        "output" => {"bandwidth"=>{"rate"=> eg_bw}, "latency"=>{"delay"=> eg }},
        "input" => {"bandwidth"=>{"rate"=> eg_bw}, "latency"=>{"delay"=> eg }}
      })
    end

    GATEWAY_VNODES.each_with_index do |node_name, idx|
      dis.viface_update(node_name, 'if0', { # gw-edge
        "output" => {"bandwidth"=>{"rate"=> eg_bw}, "latency"=>{"delay"=> eg }}, 
        "input" => {"bandwidth"=>{"rate"=> eg_bw}, "latency"=>{"delay"=> eg }}
      })
      dis.viface_update(node_name, 'if1', { # gw-gw
        "output" => {"bandwidth"=>{"rate"=> gg_bw}, "latency"=>{"delay"=> gg }}, 
        "input" => {"bandwidth"=>{"rate"=> gg_bw}, "latency"=>{"delay"=> gg }}
      })
    end

    CLIENT_VNODES.each_with_index do |node_name, idx|
      dis.viface_update(node_name, 'if0', { # cli-edge
        "output" => {"bandwidth"=>{"rate"=> el_bw}, "latency"=>{"delay"=> el }}, 
        "input" => {"bandwidth"=>{"rate"=> el_bw}, "latency"=>{"delay"=> el }}
      })
    end
end