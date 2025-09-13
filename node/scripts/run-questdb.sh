#!/bin/bash
set -e

# Variables
QUESTDB_VERSION="7.7.0"
QUESTDB_DIR="/app/questdb"
QUESTDB_PORT=9000
PG_PORT=5432

# Start QuestDB in background
echo "Starting QuestDB..."
$QUESTDB_DIR/bin/questdb.sh start
