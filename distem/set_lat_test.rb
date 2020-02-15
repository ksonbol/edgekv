require_relative 'conf'

mat = Array.new(NUM_VNODES) {Array.new(NUM_VNODES, "00")}

for r in 0..NUM_VNODES-1
  for c in 0..NUM_VNODES-1
    if r != c  # we let node-to-self latency = 0
      if (r < NUM_SERVERS) && (c < NUM_SERVERS)
        # server-to-server
        mat[r][c] = "ss"
      elsif ((r >= NUM_SERVERS) && (r < (NUM_SERVERS + NUM_GATEWAYS)) && (c < NUM_SERVERS)) ||
            ((r < NUM_SERVERS) && (c >= NUM_SERVERS) && (c < (NUM_SERVERS + NUM_GATEWAYS)))
        # server-to-gateway
        mat[r][c] = "sg"
      elsif ((r >= NUM_SERVERS) && (r < (NUM_SERVERS + NUM_GATEWAYS))) &&
            ((c >= NUM_SERVERS) && (c < (NUM_SERVERS + NUM_GATEWAYS)))
        # gateway-to-gateway
        mat[r][c] = "gg"
      elsif ((r >= (NUM_SERVERS + NUM_GATEWAYS)) && (c < NUM_SERVERS)) ||
            ((r < NUM_SERVERS) && (c >= (NUM_SERVERS + NUM_GATEWAYS)))
        # server-to-client
        mat[r][c] = "sc"
      # else: client-to-client already zeroed
      end
    end
  end
end

mat.each do |row|
    row.each do |ent|
        print "#{ent}  "
        $stdout.flush
    end
    puts ""
end