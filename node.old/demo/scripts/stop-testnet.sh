#!/bin/bash

docker ps -f name=logsdb -f name=server -f name=client -f name=node -f name=watcher -f name=signal -aq | xargs docker rm -f 
docker network rm babblenet