#!/bin/bash

if [[ ! -f cert/cacert.key ]]; then
  cd cert
  . ../make_ca.sh
  cd -
fi

if [[ -n "${SERVER_ADDR}" ]]; then
  if [[ ! -f cert/server.pem ]]; then
    echo "generate server cert for addr ${SERVER_ADDR}"
    cd cert
    . ../make_server_cert.sh goatak_server "${SERVER_ADDR}"
    cd -
  fi
  export GOATAK_SSL_MARTI=true
  export GOATAK_SSL_ENROLL=true
  export GOATAK_SSL_CA=cert/cacert.pem
  export GOATAK_SSL_CERT=cert/server.pem
  export GOATAK_SSL_KEY=cert/server.key
  export GOATAK_API_ADDR=":8443"
else
  export GOATAK_API_ADDR=":8080"
fi

./goatak_server
