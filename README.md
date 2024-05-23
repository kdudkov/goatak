# GoATAK - free ATAK/CivTAK server & client

This is fast & simple implementation of TAK server/CoT router and ATAK client with web interface.

- [Wiki](https://github.com/kdudkov/goatak/wiki)
- Support chat in Telegram: [@ru_atak](https://t.me/ru_atak)

binary builds can be downloaded
from [releases page](https://github.com/kdudkov/goatak/releases)

## Free public server

We offer a [public TAK server](https://takserver.ru) for the community free of charge. Credentials to join test group:

* address: `takserver.ru`
* set `Enroll for Client Certificate`
* user `test`
* password `111111`

iTAK connect qr-code:

![QR](img/qr.png?raw=true "QR")

## GoATAK server features

* v1 (XML) and v2 (protobuf) CoT protocol support
* certificate enrollment (v1 and v2) support
* mission packages management
* datasync / missions basic support
* user management with cli tool
* video feeds management
* visibility scopes for users (devices can communicate and see each other within one scope only)
* default preferences and maps provisioning to connected devices
* ability to log all cot's and cli utility to view cot's log and convert it to json or gpx

you can run it with docker, using `docker run -p 8088:8088 -p 8080:8080 -p 8999:8999 kdudkov/goatak_server:latest`

## Client features

* v1 (XML) and v2 (protobuf) CoT protocol support
* SSL connection support, tested with [FreeTakServer](https://github.com/FreeTAKTeam/FreeTakServer)
  , [Argustak](https://argustak.com/) and [urpc.info](https://urpc.info/)
* web-ui, ideal for big screen situation awareness center usage
* unit track - your target unit is always in the center of map
* RedX tool - to measure distance and bearing
* Digital Pointer - send DP position to all other contacts
* Add and edit units on map
* ability to log all cot's and cli utility to view cot's log and convert it to json or gpx

![Client screen](img/client.png?raw=true "Client csreen")

## Libraries used

* [Leaflet](https://leafletjs.com/)
* [Milsymbol](https://github.com/spatialillusions/milsymbol)

[![CI](https://github.com/kdudkov/goatak/actions/workflows/main.yml/badge.svg?branch=master)](https://github.com/kdudkov/goatak/actions/workflows/main.yml)

[By me a beer 🍺](https://buymeacoffee.com/kdudkov)
