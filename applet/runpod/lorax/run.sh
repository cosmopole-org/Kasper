
node index.js &

lorax-launcher --json-output --model-id mistralai/Mistral-7B-Instruct-v0.1 --port 80 --max-batch-prefill-tokens 2048 --max-input-length 2048 &

python handler.py