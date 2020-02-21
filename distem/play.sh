echo "starting etcd servers"
./server_exp.rb
echo "starting edge servers"
./edge_exp.rb
echo "starting gateway servers"
./gateway_exp.rb
echo "Running clients"
./client_exp.rb
echo "Copying YCSB files"
./ycsb_copy.rb