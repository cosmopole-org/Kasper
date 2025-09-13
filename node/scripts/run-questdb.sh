#!/bin/bash
set -e

# Variables
QUESTDB_VERSION="7.7.0"
QUESTDB_DIR="/app/questdb"
QUESTDB_PORT=9000
PG_PORT=5432

# Start QuestDB in background
echo "Starting QuestDB..."
$QUESTDB_DIR/bin/questdb.sh start &

# Wait until QuestDB is ready (port 9000)
echo "Waiting for QuestDB to start..."
until wget -q --spider http://localhost:9000; do
    echo "Waiting for QuestDB to start..."
    sleep 1
done

echo "QuestDB is ready!"
