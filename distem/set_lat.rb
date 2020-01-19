#!/usr/bin/ruby -w

require 'distem'
require_relative 'conf'

raise "Usage: ruby set_lat.rb ENVIRONMENT" unless ARGV.length == 1

  puts "Updating vnodes latencies"
  # unit is ms (millisecond)
  cc = 0.04  # cloud-to-cloud latency
  cl = 60    # cloud-to-client latency
  ee = 10    # edge-to-edge latency
  el = 10    # edge-to-client latency
  server_client = 0
  server_server = 0

  case ARGV[0]
  when "cloud"
    server_server = cc
    server_client = cl
  when "edge"
    server_server = ee
    server_client = el
  end


  latency_mat = Array.new(NUM_VNODES) {Array.new(NUM_VNODES, 0)}
  for r in 0..NUM_VNODES-1
    for c in 0..NUM_VNODES-1
      if r != c  # we let node-to-self latency = 0
        if (r < NUM_SERVERS) && (c < NUM_SERVERS)
          # server-to-server
          latency_mat[r][c] = server_server
        elsif (r >= NUM_SERVERS) ^ (c >= NUM_SERVERS)  # xor
          # server-to-client
          latency_mat[r][c] = server_client
        # else: client-to-client already zeroed
        end
      end
    end
  end

  Distem.client do |dis|
    dis.set_peers_latencies(VNODE_LIST, latency_mat)
  end
