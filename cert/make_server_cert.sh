#!/bin/bash

cert_name=server
server_name=$1
shift
names=$*

if [[ -z "$cert_name" || -z "$server_name" ]]; then
  echo "usage: make_server_cert.sh CERT_NAME SERVER_NAME [ALT_NAME...]"
  exit 1
fi

if [[ ! -e cacert.key ]]; then
  echo "No ca cert found!"
  exit 1
fi

# make server cert
openssl req -sha256 -nodes -newkey rsa:2048 -out ${cert_name}.csr -keyout ${cert_name}.key \
  -subj "/C=RU/O=${server_name}/CN=${server_name}"

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

openssl x509 -req -in ${cert_name}.csr -CA cacert.pem -CAkey cacert.key -CAcreateserial -out ${cert_name}.pem -days 3650 \
  -extfile ext.cfg
rm ${cert_name}.csr ext.cfg
