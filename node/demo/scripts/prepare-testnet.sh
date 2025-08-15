#!/bin/bash

docker run --name=logsdb --net=host --ip=172.77.5.9 -p 9042:9042 -e MAX_HEAP_SIZE=512M -e HEAP_NEWSIZE=100M cassandra