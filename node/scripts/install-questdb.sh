#!/bin/bash
set -e

# Variables
QUESTDB_VERSION="7.7.0"
QUESTDB_DIR="/app/questdb"
QUESTDB_PORT=9000
PG_PORT=5432

# Download QuestDB if not exists
if [ ! -d "$QUESTDB_DIR" ]; then
    echo "Downloading QuestDB $QUESTDB_VERSION..."
    wget -q https://github.com/questdb/questdb/releases/download/$QUESTDB_VERSION/questdb-$QUESTDB_VERSION-linux-amd64.tar.gz -O /tmp/questdb.tar.gz
    tar -xzf /tmp/questdb.tar.gz -C /app
    mv /app/questdb-$QUESTDB_VERSION $QUESTDB_DIR
fi
