FROM ubuntu:24.04
WORKDIR /app

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update -y && apt-get upgrade -y && \
    apt-get install -y curl dpkg-dev g++ gcc git ninja-build \
    software-properties-common wget zlib1g-dev && \
    apt-get install -y cmake && \
    apt-get install -y clang-17 liblld-17-dev libpolly-17-dev llvm-17-dev && \
    apt-get clean && \
    update-alternatives --install /usr/bin/clang clang /usr/bin/clang-17 100 && \
    update-alternatives --install /usr/bin/clang++ clang++ /usr/bin/clang++-17 100 && \
    update-alternatives --install /usr/bin/llvm-strip llvm-strip /usr/bin/llvm-strip-17 100

ENV CC=/usr/bin/clang-17
ENV CXX=/usr/bin/clang++-17

RUN curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash -s -- -v 0.14.0

ENV PATH="/root/.wasmedge/bin:$PATH"
ENV LD_LIBRARY_PATH="/root/.wasmedge/lib:$LD_LIBRARY_PATH"
ENV LIBRARY_PATH="/root/.wasmedge/lib:$LIBRARY_PATH"
ENV C_INCLUDE_PATH="/root/.wasmedge/include:$C_INCLUDE_PATH"
ENV CPLUS_INCLUDE_PATH="/root/.wasmedge/include:$CPLUS_INCLUDE_PATH"
ENV WASMEDGE_LIB_DIR="/root/.wasmedge/lib"

RUN apt install -y libgflags-dev libsnappy-dev libbz2-dev liblz4-dev libnuma-dev
RUN wget https://github.com/facebook/rocksdb/archive/refs/tags/v10.0.1.tar.gz
RUN tar -xvf v10.0.1.tar.gz && mv rocksdb-10.0.1 rocksdb
RUN cd rocksdb && mkdir build
WORKDIR /app/rocksdb/build
RUN cmake .. -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DWITH_SNAPPY=ON -DWITH_ZSTD=ON -DWITH_LZ4=ON -DWITH_BZ2=ON -DWITH_STATIC=ON
WORKDIR /app/rocksdb
RUN make install

WORKDIR /app

RUN apt install -y imagemagick

RUN apt install -y golang
ENV GOPROXY=https://goproxy.io,direct
ENV DOCKER_API_VERSION=1.39

COPY . .

RUN bash scripts/install-fcvmm.sh
RUN bash scripts/install-questdb.sh
RUN apt install -y postgresql-client

RUN mkdir /app/storage

RUN CGO_ENABLED=1 go build -o kasper .

ENTRYPOINT ["bash", "/app/scripts/run.sh"]
