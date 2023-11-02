#!/bin/bash

ver=$(git describe --always --tags --dirty)
branch=$(git rev-parse --symbolic-full-name --abbrev-ref HEAD)

docker build . --build-arg branch=$branch --build-arg commit=$ver -t kdudkov/goatak_server:$ver -t kdudkov/goatak_server:latest

echo "$ver"

if [[ $ver != *-* ]]; then
  echo "pushing $ver"
  docker push kdudkov/goatak_server:$ver
fi

echo "pushing latest"
docker push kdudkov/goatak_server:latest
