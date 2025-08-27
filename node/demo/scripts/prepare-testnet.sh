#!/bin/bash

docker network create kasper

docker run --name=logsdb --net=kasper --ip=172.18.0.9 -p 9042:9042 -e MAX_HEAP_SIZE=512M -e HEAP_NEWSIZE=100M cassandra
