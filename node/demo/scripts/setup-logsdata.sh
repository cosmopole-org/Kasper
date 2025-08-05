#!/bin/bash

echo "Creating keyspace"

docker exec -it logsdb cqlsh 172.77.5.9 9042 -e "create keyspace kasper with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };"
docker exec -it logsdb cqlsh 172.77.5.9 9042 -e "create table kasper.storage(id UUID, point_id text, user_id text, data text, time bigint, PRIMARY KEY(id));"
docker exec -it logsdb cqlsh 172.77.5.9 9042 -e "create index on kasper.storage(data);"