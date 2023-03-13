#!/bin/bash

ca_name=my_ca
storepass=atakatak

if [[ -e cacert.key ]]; then
  echo "ca exists!"
  exit 1
fi

openssl req -x509 -sha256 -extensions v3_ca -nodes -newkey rsa:4096 -days 3650 -out cacert.pem -keyout cacert.key \
  -subj "/C=RU/O=${ca_name}/CN=${ca_name}" \
  -addext "keyUsage = critical,cRLSign,keyCertSign"

[[ -e truststore.p12 ]] && rm truststore.p12
openssl pkcs12 -export -nokeys -name ca -in cacert.pem -out truststore.p12 -passout pass:${storepass}