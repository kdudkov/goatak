#!/bin/bash

ver=$(git describe --always --dirty)
docker build . -t kdudkov/goatak_server:$ver -t kdudkov/goatak_server:latest
docker push kdudkov/goatak_server
