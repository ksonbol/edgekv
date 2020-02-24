require 'distem'

num_gw = 1000
num_cli = 1

gw_nodes = []
(1..num_gw).each do |i|
    gw_nodes << "gw-#{i}"
end

cli_nodes = []
(1..num_cli).each do |i|
    cli_nodes << "cli-#{i}"
end

nodes = gw_nodes + cli_nodes
delay = "5ms"

Distem.client do |dis|
    nodes.each do |node|
        dis.viface_update(node, 'if0',{
            "output" => {"latency"=>{"delay"=> delay}},
            "input" => {"latency"=>{"delay"=> delay}}
          }
        )
    end
end