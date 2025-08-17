FROM python:3.13

WORKDIR /app

RUN pip install mcp-use
RUN apt update && apt install -y nodejs npm
RUN apt install -y redis-server
RUN pip install "fastapi[all]" uvicorn

COPY . .

RUN mkdir /app/input
RUN touch /app/output

ENTRYPOINT [ "bash", "run.sh" ]
