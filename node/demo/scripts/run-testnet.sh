#!/bin/bash

set -eux

N=${1:-3}
FASTSYNC=${2:-false}
WEBRTC=${3:-false}
MPWD=$(pwd)

docker network create \
  --driver=bridge \
  --subnet=172.77.0.0/16 \
  --ip-range=172.77.0.0/16 \
  --gateway=172.77.5.254 \
  babblenet

# echo "Creating keyspace"

# docker exec -it logsdb cqlsh 172.77.5.9 9042 -e "create keyspace kasper with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };"
# docker exec -it logsdb cqlsh 172.77.5.9 9042 -e "create table kasper.storage(id UUID, point_id text, user_id text, data text, PRIMARY KEY(id));"
# docker exec -it logsdb cqlsh 172.77.5.9 9042 -e "create index on kasper.storage(data);"

if "$WEBRTC"; then
    # Start the signaling server (necessary with webrtc transport). The volume
    # option copies the certificate and key files necessary to secure TLS 
    # connections
    docker run -d \
    --name=signal \
    --net=babblenet \
    --ip=172.77.15.1 \
    --volume "$(pwd)"/../src/net/signal/wamp/test_data:/signal \
    -it \
    mosaicnetworks/signal:latest --cert-file="/signal/cert.pem" --key-file="/signal/key.pem"
fi

for i in $(seq 1 $N)
do
    docker run -d --name=client$i --net=babblenet --ip=172.77.10.$i -it mosaicnetworks/dummy:latest \
    --name="client $i" \
    --client-listen="172.77.10.$i:1339" \
    --proxy-connect="172.77.5.$i:1338" \
    --discard \
    --log="debug" 
done

for i in $(seq 1 $N)
do
    docker create --name=node$i \
    --net=babblenet \
    --ip=172.77.5.$i \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --mount type=bind,source=/home/keyhan/Desktop/data$i,target=/home/ubuntu/storage \
    --privileged \
    --device /dev/kvm \
    -v /lib/modules:/lib/modules \
    -v /boot:/boot \
    kasper:latest \
    --heartbeat=100ms \
    --moniker="node$i" \
    --cache-size=50000 \
    --listen="172.77.5.$i:1337" \
    --proxy-listen="172.77.5.$i:1338" \
    --client-connect="172.77.10.$i:1339" \
    --service-listen="172.77.5.$i:80" \
    --sync-limit=100 \
    --fast-sync=$FASTSYNC \
    --log="debug" \
    --webrtc=$WEBRTC \
    --signal-addr="172.77.15.1:2443"

    # --store \
    # --bootstrap \
    # --suspend-limit=100 \
    
    
    docker start node$i
done