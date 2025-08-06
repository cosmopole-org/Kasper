FROM golang:1.24

WORKDIR /app

COPY . .

RUN mkdir /app/input
RUN touch /app/output

WORKDIR /app/app/storage/machines/7@global/gdrive
RUN go mod tidy
RUN go build -o gdrive

RUN echo hello

ENTRYPOINT [ "bash", "/app/app/storage/machines/7@global/gdrive/run.sh" ]
