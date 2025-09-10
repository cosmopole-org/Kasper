#!/bin/bash

storageRoot=${1:-"/home/keyhan/data"}
workchainId=${2:-"main"}
shardchainId=${3:-"shard-main"}
isHead=${4:-"true"}

dest="${storageRoot}/chains/${workchainId}/${shardchainId}"

# Create new key-pair and place it in new conf directory
mkdir -p $dest

# clone crypto keys
cp /root/.babble/key.pub $dest/key.pub
cp /root/.babble/priv_key $dest/priv_key

trueVal="true"
if [ "$isHead" = "$trueVal" ]; then
    cp /root/.babble/peers.genesis.json $dest/peers.genesis.json
    cp /root/.babble/peers.json $dest/peers.json
else
    # get genesis.peers.json
    echo "Fetching peers.genesis.json from node1"
    curl --trace dump -H "Work-Chain-Id: ${workchainId}" -H "Shard-Chain-Id: ${shardchainId}" -s http://api.kproto.app:8079/genesispeers > $dest/peers.genesis.json
    # get up-to-date peers.json
    echo "Fetching peers.json from node1"
    curl --trace dump -H "Work-Chain-Id: ${workchainId}" -H "Shard-Chain-Id: ${shardchainId}" -s http://api.kproto.app:8079/peers > $dest/peers.json
fi
