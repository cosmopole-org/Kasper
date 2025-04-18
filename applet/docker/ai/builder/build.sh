#!/bin/bash

token=$1
machineId=$2

rm $(pwd)/../temp.txt
cp $(pwd)/../Dockerfile $(pwd)/Dockerfile
node index.js $machineId
rm $(pwd)/Dockerfile
cp $(pwd)/temp.txt $(pwd)/../temp.txt

curl --location '172.77.5.1:8080/machines/deploy' \
--header "token: $token" \
--header 'layer: 1' \
--header 'Content-Type: application/json' \
-d @temp.txt

rm $(pwd)/temp.txt
