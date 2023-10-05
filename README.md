# GoATAK - free ATAK/CivTAK server & web-based client

This is fast & simple implementation of ATAK server/CoT router and ATAK client with web interface.

tg group: https://t.me/ru_atak

binary builds can be downloaded
from [releases page](https://github.com/kdudkov/goatak/releases)

![Alt text](client.png?raw=true "Title")

## GoATAK server features

* v1 (XML) and v2 (protobuf) CoT protocol support
* certificate enrollment (v1 and v2) support
* user management with command-line tool
* mission packs management
* video feeds management
* visibility scopes for users
* default preferences and maps provisioning to connected devices

## Web-based client features

* v1 (XML) and v2 (protobuf) CoT protocol support
* SSL connection support, tested with [FreeTakServer](https://github.com/FreeTAKTeam/FreeTakServer)
  , [Argustak](https://argustak.com/) and [urpc.info](https://urpc.info/)
* web-ui, ideal for big screen situation awareness center usage
* unit track - your target unit is always in the center of map
* RedX tool - to measure distance and bearing
* Digital Pointer - send DP position to all other contacts
* Add and edit units on map

## Test server

* address: `takserver.ru`
* set `Enroll for Client Certificate` marked
* user `test`
* password `111111`

[Wiki](https://github.com/kdudkov/goatak/wiki)

## Libraries used

* [Leaflet](https://leafletjs.com/)
* [Milsymbol](https://github.com/spatialillusions/milsymbol)

[![CI](https://github.com/kdudkov/goatak/actions/workflows/main.yml/badge.svg?branch=master)](https://github.com/kdudkov/goatak/actions/workflows/main.yml)

[By me a beer 🍺](https://buymeacoffee.com/kdudkov)
