#!/bin/bash

docker create -p 8077:8079 -p 8078:8080 -p 8079:8081 -p 8080:8082 --name=node1 \
    --ulimit nofile=65535:65535 \
    --net=host \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --mount type=bind,source=/home/keyhan/certs,target=/app/certs \
    --mount type=bind,source=/home/keyhan/data,target=/app/storage \
    --privileged \
    --device /dev/kvm \
    -v /lib/modules:/lib/modules \
    -v /boot:/boot \
    kasper:latest    
    
docker start node1