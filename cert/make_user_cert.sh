#!/bin/bash

. "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/params.sh"

user=$1

if [[ -z "$user" ]]; then
  echo "usage: $0 username"
  exit 1
fi
if [[ ! -e ca.key ]]; then
	echo "No ca cert found!"
	exit 1
fi

cat >ext.cfg <<-EOT
basicConstraints=critical,CA:FALSE
keyUsage=critical, digitalSignature, keyEncipherment
extendedKeyUsage = critical, clientAuth
EOT

# make client cert
openssl req -sha256 -nodes -newkey rsa:2048 -out ${user}.csr -keyout ${user}.key \
 -subj "/O=${user}/CN=${user}"
openssl x509 -req -in ${user}.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out ${user}.pem \
  -days 1024 -extfile ext.cfg

# make client .p12
openssl pkcs12 -export -name client-cert -in ${user}.pem -inkey ${user}.key -out ${user}.p12 -CAfile ca.pem \
  -passin pass:${PASS} -passout pass:${PASS}

rm ${user}.csr ${user}.key ${user}.pem ext.cfg
