#!/bin/bash

. "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/params.sh"

if [[ -e ${CA_NAME}.key ]]; then
  echo "ca exists!"
  exit 1
fi

openssl req -x509 -sha256 -extensions v3_ca -nodes -newkey rsa:4096 -days 3650 -out ${CA_NAME}.pem -keyout ${CA_NAME}.key \
  -subj "${SUBJ}/CN=${CA_NAME}." \
  -addext "basicConstraints = critical,CA:TRUE" \
  -addext "keyUsage = critical,cRLSign,keyCertSign"

openssl x509 -in ${CA_NAME}.pem -addtrust clientAuth -addtrust serverAuth -setalias ${CA_NAME}. -out ca-trusted.pem

#[[ -e truststore.p12 ]] && rm truststore.p12
#openssl pkcs12 -export -nokeys -name ${CA_NAME}. -in ${CA_NAME}.pem -out truststore.p12 -passout pass:${PASS}