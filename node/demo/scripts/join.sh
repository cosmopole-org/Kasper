#!/bin/bash

N=${1:-5}
FASTSYNC=${2:-false}
WEBRTC=${3:-false}
DEST=${4:-"$PWD/conf"}

dest=$DEST/node$N

# Create new key-pair and place it in new conf directory
mkdir -p $dest
echo "Generating key pair for node$N"
go run ../..//keygen/keygen.go

# get genesis.peers.json
echo "Fetching peers.genesis.json from node1"
curl -s http://165.232.32.106:80/genesispeers > $dest/peers.genesis.json

# get up-to-date peers.json
echo "Fetching peers.json from node1"
curl -s http://172.77.5.1:80/peers > $dest/peers.json

cp $dest /root/.babble

bash run-testnet.sh
