#!/bin/bash
rm $(pwd)/../temp.txt
docker run --rm --mount type=bind,source=$(pwd)/..,target=/app tinygobuild
mv $(pwd)/../main.wasm $(pwd)/main.wasm
node index.js
rm $(pwd)/main.wasm
cp $(pwd)/temp.txt $(pwd)/../temp.txt

curl --location 'localhost:8080/machines/deploy' \
--header 'token: 0bd6d7db-9399-4750-8dc2-509090185ea5-20bbcc3e-e00f-4eca-9cb3-49da39d950f2' \
--header 'layer: 1' \
--header 'Content-Type: application/json' \
-d @temp.txt

rm $(pwd)/temp.txt
