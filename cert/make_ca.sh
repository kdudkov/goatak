#!/bin/bash

CA_NAME=${CA_NAME:-root_ca}
storepass=atakatak

if [[ -e cacert.key ]]; then
  echo "ca exists!"
  exit 1
fi

openssl req -x509 -sha256 -extensions v3_ca -nodes -newkey rsa:4096 -days 3650 -out cacert.pem -keyout cacert.key \
  -subj "/C=RU/ST=RU/L=XX/OU=Goatak/CN=${CA_NAME}" \
  -addext "keyUsage = critical,cRLSign,keyCertSign" \
  -addext "basicConstraints = critical,CA:TRUE"

#openssl x509 -in cacert.pem -addtrust clientAuth -addtrust serverAuth -setalias ${CA_NAME} -out ca-trusted.pem

[[ -e truststore.p12 ]] && rm truststore.p12
openssl pkcs12 -export -nokeys -name ${CA_NAME} -in cacert.pem -out truststore.p12 -passout pass:${storepass}