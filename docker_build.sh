#!/bin/bash

image="ghcr.io/kdudkov/goatak_server"
#image="kdudkov/goatak_server"
ver=$(git describe --always --tags --dirty)
branch=$(git rev-parse --symbolic-full-name --abbrev-ref HEAD)

docker build . --build-arg branch=$branch --build-arg commit=${ver} -t ${image}:${ver} -t ${image}:latest

echo "${ver}"

if [[ ${ver} != *-* ]]; then
  echo "pushing ${ver}"
  docker push ${image}:${ver}
fi

echo "pushing latest"
docker push ${image}:latest
