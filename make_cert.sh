#!/bin/bash

server_url=192.168.0.10
server_name=test_server
user=kott
storepass=111111

if [[ ! -e server.key ]]; then
openssl req -x509 -sha256 -nodes -newkey rsa:2048 -days 1024 -out server.pem -keyout server.key -subj "/CN=${server_url}/C=RU/ST=SPb/L=SPb/O=${server_name}"
fi

openssl req -x509 -sha256 -nodes -newkey rsa:2048 -days 1024 -out client.pem -keyout client.key -subj "/CN=${user}/C=RU/ST=SPb/L=SPb/O=${user}"
openssl pkcs12 -export -name client-cert -in client.pem -inkey client.key -out client.p12 -passout pass:${storepass}

[[ -e truststore.p12 ]] && rm truststore.p12

keytool -import -alias server-cert -file server.pem -keystore truststore.p12 -storepass ${storepass} -trustcacerts -noprompt -storetype pkcs12
keytool -import -alias client-cert -file client.pem -keystore truststore.p12 -storepass ${storepass} -trustcacerts -noprompt -storetype pkcs12