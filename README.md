# GoATAK - free ATAK/CivTAK server & web-based client

Is is early alfa now. If you need production ready ATAK server - take a look
at [FreeTakServer](https://github.com/FreeTAKTeam/FreeTakServer) or [Taky](https://github.com/tkuester/taky).

This is Golang implementation of ATAK server/CoT router aimed to test some ideas about CoT message routing.

binary builds can be downloaded
from [actions](https://github.com/kdudkov/goatak/actions?query=is%3Acompleted+workflow%3ACI)

![Alt text](client.png?raw=true "Title")

## Web-based client features

* v1 (XML) and v2 (protobuf) CoT protocol support
* SSL connection support, tested with [FreeTakServer](https://github.com/FreeTAKTeam/FreeTakServer)
  and [Argustak](https://argustak.com/)
* web-ui, ideal for big screen situation awareness center usage
* unit track - your target unit is always in the center of map
* RedX tool - to measure distance and bearing
* Digital Pointer - send DP position to all other contacts

## GoATAK server features

* same features as client, but... with embedded server!

## Web client setup

1. Download latest binary build
   from [actions](https://github.com/kdudkov/goatak/actions?query=is%3Acompleted+workflow%3ACI)
1. Unzip it to local directory
1. edit `goatak_client.yml` (default values are for community server).
1. run `webclient`
1. open [http://localhost:8080](http://localhost:8080) in your browser

You can use as many config files as you want and run with specific config with `webclient -config <your_config.yml>`

### Web client config examples

simple config to connect to [Argustak](https://argustak.com/) cloud based TAK server:

```yaml
---
server_address: argustak.com:4444:ssl
web_port: 8080
me:
   callsign: username
   uid: auto
   type: a-f-G-U-C
   team: Blue
   role: Team Member
   lat: 0
   lon: 0
ssl:
   cert: username.p12
   password: password
```

## Libraries used

* [Leaflet](https://leafletjs.com/)
* [Milsymbol](https://github.com/spatialillusions/milsymbol)

[![CI](https://github.com/kdudkov/goatak/actions/workflows/main.yml/badge.svg?branch=master)](https://github.com/kdudkov/goatak/actions/workflows/main.yml)

[By me a beer 🍺](https://buymeacoffee.com/kdudkov)
