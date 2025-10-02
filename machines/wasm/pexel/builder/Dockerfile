FROM docker.arvancloud.ir/golang:1.22
WORKDIR /app

ENV DEBIAN_FRONTEND=noninteractive
ENV GOPROXY=https://goproxy.io,direct

RUN wget https://github.com/tinygo-org/tinygo/releases/download/v0.37.0/tinygo_0.37.0_amd64.deb
RUN dpkg -i tinygo_0.37.0_amd64.deb

ENTRYPOINT [ "bash", "/app/builder/tinygo.sh" ]
