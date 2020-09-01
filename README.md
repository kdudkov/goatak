# GoATAK - free ATAK/CivTAK server & web client

Is is early alfa now.
Supports TCP/UDP socket listening, events logging.

binary builds can be downloaded from [actions](https://github.com/kdudkov/goatak/actions?query=is%3Acompleted+workflow%3ACI)

## Web client setup
1. Download latest binary build from [actions](https://github.com/kdudkov/goatak/actions?query=is%3Acompleted+workflow%3ACI) (choose `web client windows` or `web client linux`)
1. Unzip it to local directory
1. edit `atak-web.yml`, see all options in [atak-web.yml.example](atak-web.yml.example)
1. run `webclient`
1. open [http://localhost:8080](http://localhost:8080) in your browser

## Libraries used:
* [Leaflet](https://leafletjs.com/)
* [Milsymbol](https://github.com/spatialillusions/milsymbol)