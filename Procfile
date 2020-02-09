# Use goreman to run `go get github.com/mattn/goreman`
# Change the path of etcd if etcd is located elsewhere
# to run this file: goreman -f Procfile start

# Local data nodes
etcd1: etcd --name infra1 --listen-client-urls http://127.0.0.1:2379 --advertise-client-urls http://127.0.0.1:2379 --listen-peer-urls http://127.0.0.1:12380 --initial-advertise-peer-urls http://127.0.0.1:12380 --initial-cluster-token local-cluster --initial-cluster 'infra1=http://127.0.0.1:12380,infra2=http://127.0.0.1:22380,infra3=http://127.0.0.1:32380' --initial-cluster-state new --enable-pprof --logger=zap --log-outputs=stderr
etcd2: etcd --name infra2 --listen-client-urls http://127.0.0.1:22379 --advertise-client-urls http://127.0.0.1:22379 --listen-peer-urls http://127.0.0.1:22380 --initial-advertise-peer-urls http://127.0.0.1:22380 --initial-cluster-token local-cluster --initial-cluster 'infra1=http://127.0.0.1:12380,infra2=http://127.0.0.1:22380,infra3=http://127.0.0.1:32380' --initial-cluster-state new --enable-pprof --logger=zap --log-outputs=stderr
etcd3: etcd --name infra3 --listen-client-urls http://127.0.0.1:32379 --advertise-client-urls http://127.0.0.1:32379 --listen-peer-urls http://127.0.0.1:32380 --initial-advertise-peer-urls http://127.0.0.1:32380 --initial-cluster-token local-cluster --initial-cluster 'infra1=http://127.0.0.1:12380,infra2=http://127.0.0.1:22380,infra3=http://127.0.0.1:32380' --initial-cluster-state new --enable-pprof --logger=zap --log-outputs=stderr
#proxy: bin/etcd grpc-proxy start --endpoints=127.0.0.1:2379,127.0.0.1:22379,127.0.0.1:32379 --listen-addr=127.0.0.1:23790 --advertise-client-url=127.0.0.1:23790 --enable-pprof

# A learner node can be started using Procfile.learner

# Global data nodes
etcd4: etcd --name infra4 --listen-client-urls http://127.0.0.1:3379 --advertise-client-urls http://127.0.0.1:3379 --listen-peer-urls http://127.0.0.1:13380 --initial-advertise-peer-urls http://127.0.0.1:13380 --initial-cluster-token global-cluster --initial-cluster 'infra4=http://127.0.0.1:13380,infra5=http://127.0.0.1:23380,infra6=http://127.0.0.1:33380' --initial-cluster-state new --enable-pprof --logger=zap --log-outputs=stderr
etcd5: etcd --name infra5 --listen-client-urls http://127.0.0.1:23379 --advertise-client-urls http://127.0.0.1:23379 --listen-peer-urls http://127.0.0.1:23380 --initial-advertise-peer-urls http://127.0.0.1:23380 --initial-cluster-token global-cluster --initial-cluster 'infra4=http://127.0.0.1:13380,infra5=http://127.0.0.1:23380,infra6=http://127.0.0.1:33380' --initial-cluster-state new --enable-pprof --logger=zap --log-outputs=stderr
etcd6: etcd --name infra6 --listen-client-urls http://127.0.0.1:33379 --advertise-client-urls http://127.0.0.1:33379 --listen-peer-urls http://127.0.0.1:33380 --initial-advertise-peer-urls http://127.0.0.1:33380 --initial-cluster-token global-cluster --initial-cluster 'infra4=http://127.0.0.1:13380,infra5=http://127.0.0.1:23380,infra6=http://127.0.0.1:33380' --initial-cluster-state new --enable-pprof --logger=zap --log-outputs=stderr

