#!/bin/bash

server_host=test.planb
server_ip=192.168.1.10

if [[ ! -e cacert.key ]]; then
	echo "No ca cert found!"
	exit 1
fi


# make server cert
openssl req -sha256 -nodes -newkey rsa:2048 -out server.csr -keyout server.key \
  -subj "/C=RU/O=${server_host}/CN=${server_host}" \
  -addext "subjectAltName=DNS:${server_host},IP:${server_ip}"

openssl x509 -req -in server.csr -CA cacert.pem -CAkey cacert.key -CAcreateserial -out server.pem -days 3650 \
 -extfile <(printf "subjectAltName=%s,%s\nextendedKeyUsage = serverAuth" "DNS:${server_host}" "IP:${server_ip}")
rm server.csr
