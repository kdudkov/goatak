#!/bin/bash

ver=$(git describe --always --tags --dirty)
docker build . -t kdudkov/goatak_server:$ver -t kdudkov/goatak_server:latest

echo "$ver"

if [[ $ver != *-* ]]; then
  echo "pushing $ver"
  docker push kdudkov/goatak_server:$ver
fi

echo "pushing latest"
docker push kdudkov/goatak_server:latest
