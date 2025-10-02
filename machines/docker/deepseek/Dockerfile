FROM docker.arvancloud.ir/ollama/ollama

WORKDIR /app

RUN mkdir /app/input

RUN apt-get -y update

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get install -y nodejs npm

ENV OLLAMA_MODELS=/root/.ollama/models

RUN touch /app/build.sh
RUN echo $' #!/bin/bash\n\
    echo "Starting Ollama server..."\n\
    ollama serve &\n\
    sleep 8\n\
    ollama pull deepseek-r1 \n\
    ' >> /app/build.sh

RUN bash /app/build.sh

RUN cd /app/input && npm -y init && npm install node-fetch@2 && npm install sqlite3

ENTRYPOINT [ "bash", "/app/input/run.sh" ]
