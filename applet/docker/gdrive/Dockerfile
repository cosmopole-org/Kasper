FROM golang:1.24

WORKDIR /app

COPY . .
RUN go mod tidy
RUN go build -o gdrive

RUN mkdir /app/input

ENTRYPOINT [ "bash", "/app/run.sh" ]
