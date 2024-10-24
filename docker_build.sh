#!/bin/bash

image="ghcr.io/kdudkov/goatak_server"
#image="kdudkov/goatak_server"
ver=$(git describe --always --tags --dirty)
branch=$(git branch --show-current)

docker build . --build-arg branch=$branch --build-arg commit=${ver} -t ${image}:${ver} -t ${image}:latest

echo "${ver}"

if [[ ${ver} != *-* ]]; then
  echo "pushing ${ver}"
  docker push ${image}:${ver}
fi

if [[ ${ver} != *-dirty ]]; then
  echo "pushing latest"
  docker push ${image}:latest
fi
