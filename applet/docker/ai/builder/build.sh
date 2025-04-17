#!/bin/bash
rm $(pwd)/../temp.txt
cp $(pwd)/../Dockerfile $(pwd)/Dockerfile
node index.js
rm $(pwd)/Dockerfile
cp $(pwd)/temp.txt $(pwd)/../temp.txt

curl --location '172.77.5.1:8080/machines/deploy' \
--header 'token: becfa7ad-13d4-43b7-8038-25079ba3587e-d65a4680-73e3-4878-8476-a3e2ee7d13e3' \
--header 'layer: 1' \
--header 'Content-Type: application/json' \
-d @temp.txt

rm $(pwd)/temp.txt
