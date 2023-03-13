#!/bin/bash

user=$1
storepass=atakatak

if [[ -z "$user" ]]; then
  echo "usage: make_user_cert.sh username"
  exit 1
fi
if [[ ! -e cacert.key ]]; then
	echo "No ca cert found!"
	exit 1
fi

# make client cert
openssl req -sha256 -nodes -newkey rsa:2048 -out ${user}.csr -keyout ${user}.key \
 -subj "/CN=${user}/O=${user}"
openssl x509 -req -in ${user}.csr -CA cacert.pem -CAkey cacert.key -CAcreateserial -out ${user}.pem -days 1024 -extfile <(echo "extendedKeyUsage = clientAuth")

# make client .p12
openssl pkcs12 -export -name client-cert -in ${user}.pem -inkey ${user}.key -out ${user}.p12 -passout pass:${storepass}

rm ${user}.csr ${user}.key ${user}.pem
