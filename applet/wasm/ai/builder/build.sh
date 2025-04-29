#!/bin/bash

token=$1
machineId=$2

rm $(pwd)/../temp.txt
docker run --rm --mount type=bind,source=$(pwd)/..,target=/app tinygobuild
mv $(pwd)/../main.wasm $(pwd)/main.wasm
