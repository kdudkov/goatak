#!/bin/bash

server_name=test_server
server_host=127.0.0.1
storepass=111111

if [[ -e ca.key ]]; then
  echo "Server key (ca.key) exists!"
  exit 1
fi

rm -f ca.pem
openssl req -x509 -sha256 -nodes -newkey rsa:2048 -days 1024 -out ca.pem -keyout ca.key -subj "/CN=${server_host}/C=US/ST=CA/O=${server_name}" -addext "subjectAltName=DNS:${server_host},IP:127.0.0.1"

# make truststore.p12
openssl pkcs12 -export -nokeys -name server-cert -in ca.pem -out truststore.p12 -passout pass:${storepass}