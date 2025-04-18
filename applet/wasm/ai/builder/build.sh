#!/bin/bash

token=$1
machineId=$2

rm $(pwd)/../temp.txt
docker run --rm --mount type=bind,source=$(pwd)/..,target=/app tinygobuild
mv $(pwd)/../main.wasm $(pwd)/main.wasm
node index.js $machineId
rm $(pwd)/main.wasm
cp $(pwd)/temp.txt $(pwd)/../temp.txt

curl --location '172.77.5.1:8080/machines/deploy' \
--header "token: $token" \
--header 'layer: 1' \
--header 'Content-Type: application/json' \
-d @temp.txt

rm $(pwd)/temp.txt
