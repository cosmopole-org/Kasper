#!/bin/bash

docker create -p 3000:3000 -p 8074:8074 -p 8076:8076 -p 8077:8077 -p 8078:8078 -p 8079:8079 --name=node1 \
    --ulimit nofile=65535:65535 \
    --net=kasper \
    --ip=10.10.0.3 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --mount type=bind,source=/home/keyhan/certs,target=/app/certs \
    --mount type=bind,source=/home/keyhan/data,target=/app/storage \
    --privileged \
    --device /dev/kvm \
    -v /lib/modules:/lib/modules \
    -v /boot:/boot \
    kasper:latest    
    
docker start node1
