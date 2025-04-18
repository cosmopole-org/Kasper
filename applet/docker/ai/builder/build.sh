#!/bin/bash
rm $(pwd)/../temp.txt
cp $(pwd)/../Dockerfile $(pwd)/Dockerfile
node index.js
rm $(pwd)/Dockerfile
cp $(pwd)/temp.txt $(pwd)/../temp.txt

curl --location '172.77.5.1:8080/machines/deploy' \
--header 'token: c1e9e98d-2a0c-40df-a279-d59c63745faa-b308ccdf-920d-41f2-93da-04fb1b7d4d06' \
--header 'layer: 1' \
--header 'Content-Type: application/json' \
-d @temp.txt

rm $(pwd)/temp.txt
