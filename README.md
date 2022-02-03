# GoATAK - free ATAK/CivTAK server & web client

Is is early alfa now. If you need production ready ATAK server - take a look
at [FreeTakServer](https://github.com/FreeTAKTeam/FreeTakServer).

This is Golang implementation of ATAK server/CoT router aimed to test some ideas about CoT message routing.

binary builds can be downloaded
from [actions](https://github.com/kdudkov/goatak/actions?query=is%3Acompleted+workflow%3ACI)

## GoATAK server features

* v1 (XML) and v2 (protobuf) CoT protocol support
* TCP, SSL and UDP (broadcast) listener
* simple web dashboard
* data package support
* easy to start - just edit config and run binary

## Web client features

* v1 (XML) and v2 (protobuf) CoT protocol support
* SSL connection support

## Web client setup

1. Download latest binary build
   from [actions](https://github.com/kdudkov/goatak/actions?query=is%3Acompleted+workflow%3ACI)
1. Unzip it to local directory
1. edit `goatak_client.yml` (default values are for community server).
1. run `webclient`
1. open [http://localhost:8080](http://localhost:8080) in your browser

You can use as many config files as you want and run with specific config with `webclient -config <your_config.yml>`

## Libraries used

* [Leaflet](https://leafletjs.com/)
* [Milsymbol](https://github.com/spatialillusions/milsymbol)

[![CI](https://github.com/kdudkov/goatak/actions/workflows/main.yml/badge.svg?branch=master)](https://github.com/kdudkov/goatak/actions/workflows/main.yml)

[By me a beer 🍺](https://buymeacoffee.com/kdudkov)
