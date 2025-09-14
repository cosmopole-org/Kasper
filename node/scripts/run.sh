#!/bin/bash

bash /app/scripts/run-fcvmm.sh
bash /app/scripts/run-questdb.sh &

valgrind --leak-check=full --show-leak-kinds=all --verbose /app/kasper
