#!/bin/bash

echo "Creating keyspace"

docker exec -it logsdb cqlsh 172.18.0.9 9042 -e "create keyspace kasper with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };"
docker exec -it logsdb cqlsh 172.18.0.9 9042 -e "create table kasper.storage(id UUID, point_id text, user_id text, data text, time bigint, edited boolean, PRIMARY KEY(id));"
docker exec -it logsdb cqlsh 172.18.0.9 9042 -e "create index on kasper.storage(data);"
