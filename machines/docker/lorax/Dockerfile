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
RUN bash build-chain.sh

# Default command will be overridden by RunPod
ENTRYPOINT ["bash", "run.sh"]
