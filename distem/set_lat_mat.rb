#!/usr/bin/ruby -w

require 'distem'
require_relative 'conf'

raise "Usage: ruby set_lat.rb ENVIRONMENT" unless ARGV.length == 1

  puts "Updating vnodes latencies"
  # unit is ms (millisecond)
  cc = 0.05  # cloud-to-cloud latency
  cl = 50    # cloud-to-client latency
  ee = 2     # edge-to-edge latency
  el = 5     # edge-to-client latency
  eg = 2     # edge-to-gateway latency
  gg = 10    # gateway-to-gateway latency
  server_client = 0
  server_server = 0

  case ARGV[0]
  when "cloud"
    server_server = cc
    server_client = cl
    server_gateway = eg  # will not be used
    gateway_gateway = gg # will not be used
  when "edge"
    server_server = ee
    server_client = el
    server_gateway = eg
    gateway_gateway = gg
  end


  latency_mat = Array.new(NUM_VNODES) {Array.new(NUM_VNODES, 0)}
  for r in 0..NUM_VNODES-1
    for c in 0..NUM_VNODES-1
      if r != c  # we let node-to-self latency = 0
        if (r < NUM_SERVERS) && (c < NUM_SERVERS)
          # server-to-server
          latency_mat[r][c] = server_server
        elsif ((r >= NUM_SERVERS) && (r < (NUM_SERVERS + NUM_GATEWAYS)) && (c < NUM_SERVERS)) ||
              ((r < NUM_SERVERS) && (c >= NUM_SERVERS) && (c < (NUM_SERVERS + NUM_GATEWAYS)))
          # server-to-gateway
          latency_mat[r][c] = server_gateway
        elsif ((r >= NUM_SERVERS) && (r < (NUM_SERVERS + NUM_GATEWAYS))) &&
              ((c >= NUM_SERVERS) && (c < (NUM_SERVERS + NUM_GATEWAYS)))
          # gateway-to-gateway
          latency_mat[r][c] = gateway_gateway
        elsif ((r >= (NUM_SERVERS + NUM_GATEWAYS)) && (c < NUM_SERVERS)) ||
              ((r < NUM_SERVERS) && (c >= (NUM_SERVERS + NUM_GATEWAYS)))
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
