#!/bin/bash

cp nginx.conf /home/keyhan/data/docker_proxy/nginx.conf

docker run -d --name kasper-proxy \
    --network kasper --ip 10.10.0.5 -p 8082:8082 -p 8443:8443 \
    -v /home/keyhan/data/docker_proxy/nginx.conf:/etc/nginx/nginx.conf:ro \
    -v /home/keyhan/data/docker_proxy/ssl:/etc/nginx/ssl:ro \
    nginx:alpine

docker create -p 9999:9999 -p 3000:3000 -p 8074:8074 -p 8076:8076 -p 8077:8077 -p 8078:8078 -p 1337:1337 -p 8079:8000 --name=node1 \
    --ulimit nofile=65535:65535 \
    --net=kasper \
    --ip=10.10.0.3 \
    --memory=1500m
    -v /var/run/docker.sock:/var/run/docker.sock \
    --mount type=bind,source=/root/.babble,target=/root/.babble \
    --mount type=bind,source=/home/keyhan/certs,target=/app/certs \
    --mount type=bind,source=/home/keyhan/data,target=/app/storage \
    --mount type=bind,source=/home/keyhan/packets,target=/app/questdb/db \
    --privileged \
    --device /dev/kvm \
    -v /lib/modules:/lib/modules \
    -v /boot:/boot \
    kasper:latest    
    
docker start node1
