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

RUN pip install numpy
RUN pip install tensorflow-cpu==1.10.0
RUN pip install keras==2.3.0

WORKDIR /app

RUN mkdir /app/input

ENTRYPOINT [ "bash", "/app/input/run.sh" ]
