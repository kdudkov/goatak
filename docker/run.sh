#!/bin/bash

goatak_base_path="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"

mkdir -p "$goatak_base_path/cert/files"
cd "$goatak_base_path/cert/files"

if [[ ! -f ca.key ]]; then
  . "$goatak_base_path/cert/make_ca.sh"
fi

if [[ -n "${SRV_HOST}" ]]; then
  if [[ ! -f ${SRV_HOST}.pem ]]; then
    echo "generate new server cert for addr ${SRV_HOST}"
    . "$goatak_base_path/cert/make_server_cert.sh" ${SRV_HOST}
    . "$goatak_base_path/cert/make_connect_dp.sh" user
  fi

  export GOATAK_SSL_MARTI=true
  export GOATAK_SSL_ENROLL=true
  export GOATAK_SSL_CA=cert/files/ca.pem
  export GOATAK_SSL_CERT=cert/files/${SRV_HOST}.pem
  export GOATAK_SSL_KEY=cert/files/${SRV_HOST}-chain.key
  export GOATAK_API_ADDR=":8443"
  export GOATAK_TLS_ADDR=":8089"
else
  export GOATAK_API_ADDR=":8080"
fi

cd "$goatak_base_path"
echo "starting Goatak server for ${SRV_HOST}"
./goatak_server
