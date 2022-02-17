#!/bin/bash

server_name=test_server
server_host=127.0.0.1
storepass=111111

if [[ -e cacert.key ]]; then
  echo "Server key (cacert.key) exists!"
  exit 1
fi

rm -f ca.pem
openssl req -x509 -sha256 -nodes -newkey rsa:2048 -days 3650 -out cacert.pem -keyout cacert.key \
  -subj "/CN=${server_host}/C=US/ST=CA/O=${server_name}" \
  -addext "basicConstraints=critical,CA:TRUE"

[[ -e truststore.p12 ]] && rm truststore.p12

# make truststore.p12
openssl pkcs12 -export -nokeys -name ca -in cacert.pem -out truststore.p12 -passout pass:${storepass}

# make client cert
openssl req -sha256 -nodes -newkey rsa:2048 -out server.csr -keyout server.key \
  -subj "/CN=${server_host}/C=US/ST=CA/O=${server_name}" \
  -addext "subjectAltName=DNS:${server_host},IP:127.0.0.1"

openssl x509 -req -in server.csr -CA cacert.pem -CAkey cacert.key -CAcreateserial -out server.pem -days 3650
rm server.csr
