# unreleased

FEATURES:

- gpsd support for webclient

# 0.16.4

FEATURES:

- new `takreplay` format: `stats`
- new `takreplay` format: `broadcast`
- `cots_processed` metric now has labels `type` and `scope`
- new `route_pings` server config option
- some new message type names added
- new `welcome_msg` server config option
- `connections` metric now has label `scope`

FIXES:

- client npe fixed

# 0.16.3

FEATURES:

- show units/points scope in admin dashboard
- create mission notification message
- new server config option - `interscope_chat` that allows chat messages routing between scopes

FIXES:

- fixed showing two chat messages when the server broadcast your chat message back

# 0.16.2

FIXES:

- fixed chat in client
- fixed ws status in client
- fixed server dashboard

# 0.16.0

FEATURES:

- server admin map and client now work through websocket API
- video feed scope filtering
- data packages scope filtering