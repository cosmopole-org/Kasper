#!/bin/bash
set -e

# Variables
QUESTDB_DIR="/app/questdb"
export QDB_PG_ENABLED=true
export QDB_PG_PORT=5432
export QDB_PG_BIND=0.0.0.0

echo "pg.enabled=true\npg.port=5432\npg.bind=0.0.0.0">$QUESTDB_DIR/conf/server.conf

# Start QuestDB in background
echo "Starting QuestDB..."
$QUESTDB_DIR/bin/questdb.sh start -d $QUESTDB_DIR -f
