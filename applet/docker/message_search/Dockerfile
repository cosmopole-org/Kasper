FROM golang:1.24

WORKDIR /app

COPY . .

RUN mkdir /app/input
RUN touch /app/output

RUN go mod tidy
RUN go build -o searcher

ENTRYPOINT [ "/app/searcher" ]
