FROM ghcr.io/predibase/lorax:latest

# Set working directory
WORKDIR /app

# Install RunPod SDK
RUN pip install runpod

# Copy files
COPY applet/docker/lorax .
RUN ls

# Expose port for LoRAX server
EXPOSE 80

# Set environment variables
ENV CUDA_VISIBLE_DEVICES=0
ENV TRANSFORMERS_CACHE=/tmp/transformers_cache
ENV HF_HOME=/tmp/hf_home
ENV HF_TOKEN=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

# Create cache directories
RUN mkdir -p /tmp/transformers_cache /tmp/hf_home

RUN python runpod-bridge/download.py

RUN apt -y update && apt install -y curl
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.3/install.sh | bash
RUN export NVM_DIR="$HOME/.nvm"\
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"\
    [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion" &&\
    nvm install 18 && cd chain-bridge && npm i

# Default command will be overridden by RunPod
ENTRYPOINT ["bash", "run.sh"]
