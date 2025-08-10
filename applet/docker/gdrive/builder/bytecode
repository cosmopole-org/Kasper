FROM golang:1.24

WORKDIR /app

COPY . .

RUN mkdir /app/input
RUN touch /app/output

RUN echo "building... 2"

RUN go mod tidy
RUN go build -o gdrive

ENTRYPOINT [ "/app/gdrive", "--command=adminInit" ]
