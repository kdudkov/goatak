#!/bin/bash

ver=$(git describe --always --tags --dirty)
branch=$(git rev-parse --symbolic-full-name --abbrev-ref HEAD)

docker buildx build --platform linux/arm/v7,linux/arm64/v8,linux/amd64 --push . --build-arg branch=$branch --build-arg commit=$ver -t kdudkov/goatak_server:latest

