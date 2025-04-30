#!/bin/bash

. "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/params.sh"

SERVER_NAME=$1
CERT_NAME=${SERVER_NAME}
#shift
names=$*

if [[ -z "${CERT_NAME}" || -z "${SERVER_NAME}" ]]; then
  echo "usage: $0 SERVER_NAME [ALT_NAME...]"
  exit 1
fi

if [[ ! -e ca.key ]]; then
  echo "No ca cert found!"
  exit 1
fi

# make server cert
openssl req -sha256 -nodes -newkey rsa:2048 -out ${CERT_NAME}.csr -keyout ${CERT_NAME}.key \
  -subj "${SUBJ}/CN=${SERVER_NAME}"

cat >ext.cfg <<-EOT
basicConstraints=critical,CA:TRUE
keyUsage = critical,digitalSignature,keyEncipherment,cRLSign,keyCertSign
extendedKeyUsage = critical,clientAuth,serverAuth
EOT

if [[ -n "$names" ]]; then
  for d in $names; do
    [[ -n "$s" ]] && s="$s,"

    if [[ "$d" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
      s="${s}IP:${d}"
    else
      s="${s}DNS:${d}"
    fi
  done
  echo "subjectAltName=$s" >>ext.cfg
fi

cat ext.cfg

openssl x509 -req -in ${CERT_NAME}.csr -CA ca.pem -CAkey ca.key -CAcreateserial \
  -out ${CERT_NAME}.pem -days 3650 -extfile ext.cfg

rm ext.cfg ${CERT_NAME}.csr

rm ${CERT_NAME}-chain.pem
cp ${CERT_NAME}.pem ca.pem >> ${CERT_NAME}-chain.pem
