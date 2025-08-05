#!/bin/bash

docker create -p 8077:8079 -p 8078:8080 --name=node1 \
    --ulimit nofile=65535:65535 \
    --net=kasper \
    --ip=172.77.5.1 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --mount type=bind,source=/home/keyhan/data1,target=/app/storage \
    --privileged \
    --device /dev/kvm \
    -v /lib/modules:/lib/modules \
    -v /boot:/boot \
    kasper:latest    
    
docker start node1
