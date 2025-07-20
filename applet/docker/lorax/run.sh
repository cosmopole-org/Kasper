
export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion" &&
    node /app/chain-bridge/main.js &

lorax-launcher --json-output --model-id mistralai/Mistral-7B-Instruct-v0.1 --port 80 --max-batch-prefill-tokens 2048 --max-input-length 2048 &

python /app/runpod-bridge/handler.py