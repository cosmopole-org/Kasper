#!/bin/bash

docker network create \
  --driver bridge \
  --subnet 10.10.0.0/16 \
  --gateway 10.10.0.1 \
  kasper

bash build-conf.sh 1

docker run --name=logsdb --net=kasper --ip=10.10.0.9 -p 9000:9000 -p 8812:8812 -p 5432:5432 questdb/questdb

mdkir -p /home/keyhan/data/docker_proxy/ssl

openssl req -x509 -newkey ed25519 -days 3650 \
  -noenc -keyout nginx-selfsigned.key -out nginx-selfsigned.crt -subj "/CN=example.com" \
  -addext "subjectAltName=DNS:example.com,DNS:*.example.com,IP:10.0.0.1"

cp nginx-selfsigned.key /home/keyhan/data/docker_proxy/ssl/nginx-selfsigned.key
cp nginx-selfsigned.crt /home/keyhan/data/docker_proxy/ssl/nginx-selfsigned.crt
