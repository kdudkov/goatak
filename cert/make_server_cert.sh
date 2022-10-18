#!/bin/bash

server_name="testserver.cot"
server_hosts="testserver.cot 192.168.1.10 127.0.0.1"

if [[ ! -e cacert.key ]]; then
	echo "No ca cert found!"
	exit 1
fi

# make server cert
openssl req -sha256 -nodes -newkey rsa:2048 -out server.csr -keyout server.key \
  -subj "/C=RU/O=${server_name}/CN=${server_name}"

cat > ext.cfg <<- EOT
basicConstraints=critical,CA:TRUE
keyUsage = critical,digitalSignature,keyEncipherment,cRLSign,keyCertSign
extendedKeyUsage = critical,clientAuth,serverAuth
EOT

for d in $server_hosts
do
	[[ "$s" != "" ]] && s="$s,"

	if [[ "$d" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
		s="${s}IP:${d}"
	else
		s="${s}DNS:${d}"
	fi
done
echo "subjectAltName=$s" >> ext.cfg

openssl x509 -req -in server.csr -CA cacert.pem -CAkey cacert.key -CAcreateserial -out server.pem -days 3650 \
 -extfile ext.cfg
rm server.csr ext.cfg

#cat cacert.pem >> server.pem
