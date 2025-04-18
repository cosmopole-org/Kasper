FROM python:3.10-slim-buster

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    build-essential \
    libblas3 \
    liblapack3 \
    libopenblas-dev \
    liblapack-dev \
    libatlas-base-dev \
    gfortran \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN mkdir /app/input

ENTRYPOINT [ "bash", "/app/input/run.sh" ]
