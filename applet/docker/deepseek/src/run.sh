#!/bin/bash

export OLLAMA_MODELS=/root/.ollama/models
ollama serve &
sleep 8
node /app/input/run.js
