#!/bin/bash

rm $(pwd)/../temp.txt
docker run --rm --mount type=bind,source=$(pwd)/..,target=/app tinygobuild
mv $(pwd)/../main.wasm $(pwd)/bytecode
