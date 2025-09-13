#!/bin/bash

docker network create \
  --driver bridge \
  --subnet 10.10.0.0/16 \
  --gateway 10.10.0.1 \
  kasper

bash build-conf.sh 1

docker run --name=logsdb --net=kasper --ip=10.10.0.9 -p 9000:9000 -p 8812:8812 -p 5432:5432 questdb/questdb
