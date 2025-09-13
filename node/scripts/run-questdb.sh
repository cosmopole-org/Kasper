#!/bin/bash
set -e

# Variables
export QDB_PG_ENABLED=true
export QDB_PG_PORT=5432
export QDB_PG_BIND=0.0.0.0

# Start QuestDB in background
echo "Starting QuestDB..."
$QUESTDB_DIR/bin/questdb.sh start
