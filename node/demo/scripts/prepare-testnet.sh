docker network create \
  --driver=bridge \
  --subnet=172.77.0.0/16 \
  --ip-range=172.77.0.0/16 \
  --gateway=172.77.5.254 \
  babblenet

docker run --name=logsdb --net=babblenet --ip=172.77.5.9 -p 9042:9042 cassandra
