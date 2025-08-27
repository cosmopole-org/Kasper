#!/bin/bash

docker network create \
  --driver bridge \
  --subnet 10.10.0.0/16 \
  --gateway 10.10.0.1 \
  kasper

docker run --name=logsdb --net=kasper --ip=10.10.0.9 -p 9042:9042 -e MAX_HEAP_SIZE=512M -e HEAP_NEWSIZE=100M cassandra
