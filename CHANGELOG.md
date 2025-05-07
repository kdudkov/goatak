# GoATAK changelog

## v0.21.1: 2025-05-07

## v0.21.0: 2025-05-07
### Added
* Now user can see DPs in read_scope scopes
* MySQL support
* Admin device management page
* docker image auto certs generation
* admin api auth
* no more users.yml, now all information is stored in database
### Changed
* local endpoint for prometheus metrics (local_addr in config)
* client now can work standalone if no connect string in config
### Fixed
* corrupted DataPackage XML
* WinTak certs problem with marti api
* DataPackage url
* certs generation scripts
* correct scope for WebTak ws handler

## v0.20.0: 2025-04-11
### Added
* Points page in admin interface
* extended roles for atak 5.4
### Changed
* MissionPackage information is now stored in database
* database schema
* vue3 & leaflet 1.9.4
### Deprecated
* `datasync` option in config
### Fixed
* disable colored logs pn windows
* MissionChange for resources
* dp file upload

## v0.19.1: 2024-10-24
### Added
* Block user by UID (drops connection and prevent cert enrollment)
* Unread/total messages badge and different color, if unread > 0

## v0.19.0
### Added
* switch to koanf from viper
* switch to fiber from air
* smaller icons in menu
* do not log welcome message in messages log

### Fixed
* fix ws handler never removed
* fix meshtastic plugin versions
* fix metrics
* enlarge body limit for marti api
* fix error in pos message handler
* fix fiber error logging
* fix null initial handler time

## v0.18.1
### Added
* mission packages download link in admin interface
* etag header added
### Fixed
* many UI fixes
* stats format in `takreplay` app fixed

## v0.18.0
### Added
* messages counters in client now show number of unread messages
* fixed chat in webclient: message delivery, send message with enter key, etc
* `takreplay` format `stats` now shows clients and devices information
* new `cots_dropped` metric
* server config options `interscope_chat` and `route_pings` are removed
### Fixed
* fixed server bottleneck with ssl handshake

## v0.17.1
### Added
* multiple file processing with `takreplay`. Like `./takreplay -format stats data/log/*.tak`
* cot drops metric added
### Fixed
* xml cot handler fixed

## v0.17.0
### Added
* gpsd support for webclient
* mission packages storage refactoring
* new `mm` (Mission Manager) cli utility
### Fixed
* fixed some missions and mission packages issues

## v0.16.4
### Added
* new `takreplay` format: `stats`
* new `takreplay` format: `broadcast`
* `cots_processed` metric now has labels `type` and `scope`
* new `route_pings` server config option
* some new message type names added
* new `welcome_msg` server config option
* `connections` metric now has label `scope`
### Fixed
* client npe fixed

## v0.16.3
### Added
* show units/points scope in admin dashboard
* create mission notification message
* new server config option - `interscope_chat` that allows chat messages routing between scopes
### Fixed
* fixed showing two chat messages when the server broadcast your chat message back

## v0.16.2
### Fixed
* fixed chat in client
* fixed ws status in client
* fixed server dashboard

## v0.16.0
### Added
* server admin map and client now work through websocket API
* video feed scope filtering
* data packages scope filtering