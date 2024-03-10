
function needIconUpdate(oldUnit, newUnit) {
    if (oldUnit.sidc !== newUnit.sidc || oldUnit.status !== newUnit.status) return true;
    if (oldUnit.speed !== newUnit.speed || oldUnit.direction !== newUnit.direction) return true;
    if (oldUnit.team !== newUnit.team || oldUnit.role !== newUnit.role) return true;

    if (newUnit.sidc.charAt(2) === 'A' && oldUnit.hae !== newUnit.hae) return true;
    return false;
}

let app = new Vue({
    el: '#app',
    data: {
        map: null,
        layers: null,
        conn: null,
        units: new Map(),
        messages: [],
        ts: 0,
        locked_unit_uid: '',
        current_unit_uid: null,
        config: null,
        tools: new Map(),
        me: null,
        coords: null,
        point_num: 1,
        coord_format: "d",
        form_unit: {},
        types: null,
        chatroom: "",
        chat_uid: "",
        chat_msg: "",
    },

    mounted() {
        this.map = L.map('map');
        this.map.setView([60, 30], 11);

        L.control.scale({metric: true}).addTo(this.map);

        this.getConfig();

        let supportsWebSockets = 'WebSocket' in window || 'MozWebSocket' in window;

        if (supportsWebSockets) {
            this.connect();
            setInterval(this.getAllUnits, 60000);
        }

        this.renew();
        setInterval(this.renew, 30000);

        this.map.on('click', this.mapClick);
        this.map.on('mousemove', this.mouseMove);

        this.formFromUnit(null);
    },

    computed: {
        current_unit: function () {
            return this.current_unit_uid ? this.current_unit_uid && this.getCurrentUnit() : null;
        }
    },

    methods: {
        getConfig: function () {
            let vm = this;

            fetch('/config')
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    vm.config = data;

                    vm.map.setView([data.lat, data.lon], data.zoom);

                    if (vm.config.callsign) {
                        vm.me = L.marker([data.lat, data.lon]);
                        vm.me.setIcon(L.icon({
                            iconUrl: "/static/icons/self.png",
                            iconAnchor: new L.Point(16, 16),
                        }));
                        vm.me.addTo(vm.map);

                        fetch('/types')
                            .then(function (response) {
                                return response.json()
                            })
                            .then(function (data) {
                                vm.types = data;
                            });
                    }

                    layers = L.control.layers({}, null, {hideSingleBase: true});
                    layers.addTo(vm.map);

                    let first = true;
                    data.layers.forEach(function (i) {
                        let opts = {
                            minZoom: i.minZoom ?? 1,
                            maxZoom: i.maxZoom ?? 20,
                        }

                        if (i.parts) {
                            opts["subdomains"] = i.parts;
                        }

                        l = L.tileLayer(i.url, opts);

                        layers.addBaseLayer(l, i.name);

                        if (first) {
                            first = false;
                            l.addTo(vm.map);
                        }
                    });
                });
        },

        connect: function () {
            let url = (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws';
            let vm = this;

            this.getAllUnits();

            this.conn = new WebSocket(url);

            this.conn.onmessage = function (e) {
                vm.processUnit(JSON.parse(e.data));
            };

            this.conn.onopen = function (e) {
                console.log("connected");
            };

            this.conn.onerror = function (e) {
                console.log("error");
            };

            this.conn.onclose = function (e) {
                console.log("closed");
                vm.conn = null;
                setTimeout(vm.connect, 3000);
            };
        },

        getAllUnits: function () {
            let vm = this;

            fetch('/unit')
                .then(function (response) {
                    return response.json()
                })
                .then(vm.processUnits);
        },

        renew: function () {
            let vm = this;

            if (!this.conn) {
                this.getAllUnits();
            }

            fetch('/message')
                .then(function (response) {
                    return response.json();
                })
                .then(function (data) {
                    vm.messages = data;
                });

            if (this.getTool("dp1")) {
                let p = this.getTool("dp1").getLatLng();

                const requestOptions = {
                    method: "POST",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify({lat: p.lat, lon: p.lng, name: "DP1"})
                };
                fetch("/dp", requestOptions);
            }
        },

        processUnits: function (data) {
            let keys = new Set();

            for (let u of data) {
                keys.add(this.processUnit(u)?.uid);
            }

            for (const k of this.units.keys()) {
                if (!keys.has(k)) {
                    this.removeUnit(k);
                }
            }

            this.ts += 1;
        },

        processUnit: function (u) {
            if (!u) return;
            let unit = this.units.get(u.uid);
            let updateIcon = false;
            if (u.category === "delete") {
                this.removeUnit(u.uid);
                return null;
            }
            if (!unit) {
                this.units.set(u.uid, u);
                unit = u;
                updateIcon = true;
            } else {
                updateIcon = needIconUpdate(unit, u);
                for (const k of Object.keys(u)) {
                    unit[k] = u[k];
                }
            }
            this.updateMarker(unit, false, updateIcon);

            if (this.locked_unit_uid === unit.uid) {
                this.map.setView([unit.lat, unit.lon]);
            }

            return unit;
        },

        updateMarker: function (unit, draggable, updateIcon) {
            if (unit.lon === 0 && unit.lat === 0) {
                if (unit.marker) {
                    this.map.removeLayer(unit.marker);
                    unit.marker = null;
                }
                return
            }

            if (unit.marker) {
                if (updateIcon) {
                    unit.marker.setIcon(getIcon(unit, true));
                }
            } else {
                unit.marker = L.marker([unit.lat, unit.lon], {draggable: draggable});
                unit.marker.on('click', function (e) {
                    app.setCurrentUnitUid(unit.uid, false);
                });
                if (draggable) {
                    unit.marker.on('dragend', function (e) {
                        unit.lat = marker.getLatLng().lat;
                        unit.lon = marker.getLatLng().lng;
                    });
                }
                unit.marker.setIcon(getIcon(unit, true));
                unit.marker.addTo(this.map);
            }

            unit.marker.setLatLng([unit.lat, unit.lon]);
            unit.marker.bindTooltip(popup(unit));
        },

        removeUnit: function (uid) {
            if (!this.units.has(uid)) return;

            let item = this.units.get(uid);
            if (item.marker) {
                this.map.removeLayer(item.marker);
                item.marker.remove();
            }
            this.units.delete(uid);
            if (this.current_unit_uid === uid) {
                this.setCurrentUnitUid(null, false);
            }
        },

        setCurrentUnitUid: function (uid, follow) {
            if (uid && this.units.has(uid)) {
                this.current_unit_uid = uid;
                let u = this.units.get(uid);
                if (follow) this.mapToUnit(u);
                this.formFromUnit(u);
            } else {
                this.current_unit_uid = null;
                this.formFromUnit(null);
            }
        },

        getCurrentUnit: function () {
            if (!this.current_unit_uid || !this.units.has(this.current_unit_uid)) return null;
            return this.units.get(this.current_unit_uid);
        },

        byCategory: function (s) {
            let arr = Array.from(this.units.values()).filter(function (u) {
                return u.category === s
            });
            arr.sort(function (a, b) {
                let ua = a.callsign.toLowerCase(), ub = b.callsign.toLowerCase();
                if (ua < ub) return -1;
                if (ua > ub) return 1;
                return 0;
            });
            return this.ts && arr;
        },

        mapToUnit: function (u) {
            if (!u) {
                return;
            }
            if (u.lat !== 0 || u.lon !== 0) {
                this.map.setView([u.lat, u.lon]);
            }
        },

        getImg: function (item) {
            return getIconUri(item, false).uri;
        },

        milImg: function (item) {
            return getMilIcon(item, false).uri;
        },

        dt: function (str) {
            let d = new Date(Date.parse(str));
            return ("0" + d.getDate()).slice(-2) + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" +
                d.getFullYear() + " " + ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
        },

        sp: function (v) {
            return (v * 3.6).toFixed(1);
        },

        modeIs: function (s) {
            return document.getElementById(s).checked === true;
        },

        mouseMove: function (e) {
            this.coords = e.latlng;
        },

        mapClick: function (e) {
            if (this.modeIs("redx")) {
                this.addOrMove("redx", e.latlng, "/static/icons/x.png")
                return;
            }
            if (this.modeIs("dp1")) {
                this.addOrMove("dp1", e.latlng, "/static/icons/spoi_icon.png")
                return;
            }
            if (this.modeIs("point")) {
                let uid = uuidv4();
                let now = new Date();
                let stale = new Date(now);
                stale.setDate(stale.getDate() + 365);
                let u = {
                    uid: uid,
                    category: "point",
                    callsign: "point-" + this.point_num++,
                    sidc: "",
                    start_time: now,
                    last_seen: now,
                    stale_time: stale,
                    type: "b-m-p-s-m",
                    lat: e.latlng.lat,
                    lon: e.latlng.lng,
                    hae: 0,
                    speed: 0,
                    course: 0,
                    status: "",
                    text: "",
                    parent_uid: "",
                    parent_callsign: "",
                    local: true,
                    send: false,
                }
                if (this.config && this.config.uid) {
                    u.parent_uid = this.config.uid;
                    u.parent_callsign = this.config.callsign;
                }

                this.sendUnit(u);
                this.setCurrentUnitUid(u.uid, true);
            }
            if (this.modeIs("me")) {
                this.config.lat = e.latlng.lat;
                this.config.lon = e.latlng.lng;
                this.me.setLatLng(e.latlng);
                const requestOptions = {
                    method: "POST",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify({lat: e.latlng.lat, lon: e.latlng.lng})
                };
                fetch("/pos", requestOptions);
            }
        },

        sendUnit: function (u) {
            const requestOptions = {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify(this.cleanUnit(u))
            };
            let vm = this;
            fetch("/unit", requestOptions)
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    vm.processUnit(data);
                });
        },

        formFromUnit: function (u) {
            if (!u) {
                this.form_unit = {
                    callsign: "",
                    category: "",
                    type: "",
                    subtype: "",
                    aff: "",
                    text: "",
                    send: false,
                    root_sidc: null,
                };
            } else {
                this.form_unit = {
                    callsign: u.callsign,
                    category: u.category,
                    type: u.type,
                    subtype: "G",
                    aff: "h",
                    text: u.text,
                    send: u.send,
                    root_sidc: this.types,
                };

                if (u.type.startsWith('a-')) {
                    this.form_unit.type = 'b-m-p-s-m';
                    this.form_unit.aff = u.type.substring(2, 3);
                    this.form_unit.subtype = u.type.substring(4);
                    this.form_unit.root_sidc = this.getRootSidc(u.type.substring(4))
                }
            }
        },

        saveEditForm: function () {
            let u = this.getCurrentUnit();
            if (!u) return;

            u.callsign = this.form_unit.callsign;
            u.category = this.form_unit.category;
            u.send = this.form_unit.send;
            u.text = this.form_unit.text;

            if (this.form_unit.category === "unit") {
                u.type = ["a", this.form_unit.aff, this.form_unit.subtype].join('-');
                u.sidc = this.sidcFromType(u.type);
            } else {
                u.type = this.form_unit.type;
                u.sidc = "";
            }
            this.sendUnit(u);
        },

        getRootSidc: function (s) {
            let curr = this.types;

            if (!curr?.next) {
                return null;
            }

            for (; ;) {
                let found = false;
                for (const k of curr.next) {
                    if (k.code === s) {
                        return curr;
                    }

                    if (s.startsWith(k.code)) {
                        curr = k;
                        found = true;
                        break
                    }
                }
                if (!found) {
                    return null;
                }
            }
        },

        getSidc: function (s) {
            let curr = this.types;

            if (s === "") {
                return curr;
            }

            if (!curr?.next) {
                return null;
            }

            for (; ;) {
                for (const k of curr.next) {
                    if (k.code === s) {
                        return k;
                    }

                    if (s.startsWith(k.code)) {
                        curr = k;
                        break
                    }
                }
            }
            return null;
        },

        setFormRootSidc: function (s) {
            let t = this.getSidc(s);
            if (t?.next) {
                this.form_unit.root_sidc = t;
                this.form_unit.subtype = t.next[0].code;
            } else {
                this.form_unit.root_sidc = this.types;
                this.form_unit.subtype = this.types.next[0].code;
            }
        },

        removeTool: function (name) {
            if (this.tools.has(name)) {
                let p = this.tools.get(name);
                this.map.removeLayer(p);
                p.remove();
                this.tools.delete(name);
                this.ts++;
            }
        },

        getTool: function (name) {
            return this.tools.get(name);
        },

        addOrMove(name, coord, icon) {
            if (this.tools.has(name)) {
                this.tools.get(name).setLatLng(coord);
            } else {
                let p = new L.marker(coord).addTo(this.map);
                if (icon) {
                    p.setIcon(L.icon({
                        iconUrl: icon,
                        iconSize: [20, 20],
                        iconAnchor: new L.Point(10, 10),
                    }));
                }
                this.tools.set(name, p);
            }
            this.ts++;
        },

        printCoordsll: function (latlng) {
            return this.printCoords(latlng.lat, latlng.lng);
        },

        printCoords: function (lat, lng) {
            return lat.toFixed(6) + "," + lng.toFixed(6);
        },

        latlng: function (lat, lon) {
            return L.latLng(lat, lon);
        },

        distBea: function (p1, p2) {
            let toRadian = Math.PI / 180;
            // haversine formula
            // bearing
            let y = Math.sin((p2.lng - p1.lng) * toRadian) * Math.cos(p2.lat * toRadian);
            let x = Math.cos(p1.lat * toRadian) * Math.sin(p2.lat * toRadian) - Math.sin(p1.lat * toRadian) * Math.cos(p2.lat * toRadian) * Math.cos((p2.lng - p1.lng) * toRadian);
            let brng = Math.atan2(y, x) * 180 / Math.PI;
            brng += brng < 0 ? 360 : 0;
            // distance
            let R = 6371000; // meters
            let deltaF = (p2.lat - p1.lat) * toRadian;
            let deltaL = (p2.lng - p1.lng) * toRadian;
            let a = Math.sin(deltaF / 2) * Math.sin(deltaF / 2) + Math.cos(p1.lat * toRadian) * Math.cos(p2.lat * toRadian) * Math.sin(deltaL / 2) * Math.sin(deltaL / 2);
            let c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
            let distance = R * c;
            return (distance < 10000 ? distance.toFixed(0) + "m " : (distance / 1000).toFixed(1) + "km ") + brng.toFixed(1) + "°T";
        },

        contactsNum: function () {
            let online = 0;
            let total = 0;
            this.units.forEach(function (u) {
                if (u.category === "contact") {
                    if (u.status === "Online") online += 1;
                    if (u.status !== "") total += 1;
                }
            })

            return online + "/" + total;
        },

        countByCategory: function (s) {
            let total = 0;
            this.units.forEach(function (u) {
                if (u.category === s) total += 1;
            })

            return total;
        },

        msgNum: function () {
            if (!this.messages) return 0;
            let n = 0;
            for (const [key, value] of Object.entries(this.messages)) {
                if (value.messages) {
                    n += value.messages.length;
                }
            }
            return n;
        },

        msgNum1: function (k) {
            if (!this.messages || !this.messages[k].messages) return 0;
            return this.messages[k].messages.length;
        },

        setChat: function (uid, chatroom) {
            this.chat_uid = uid;
            this.chatroom = chatroom;
        },

        openChat: function (uid, chatroom) {
            this.chat_uid = uid;
            this.chatroom = chatroom;
            new bootstrap.Modal(document.getElementById('messages')).show();
        },

        getMessages: function () {
            if (!this.chat_uid) {
                return [];
            }
            return this.messages[this.chat_uid] ? this.messages[this.chat_uid].messages : [];
        },

        getUnitName: function (u) {
            let res = u.callsign || "no name";
            if (u.parent_uid === this.config.uid) {
                if (u.send === true) {
                    res = "+ " + res;
                } else {
                    res = "* " + res;
                }
            }
            return res;
        },

        cancelEditForm: function () {
            this.formFromUnit(this.getCurrentUnit());
        },

        sidcFromType: function (s) {
            if (!s.startsWith('a-')) return "";

            let n = s.split('-');

            let sidc = 'S' + n[1];

            if (n.length > 2) {
                sidc += n[2] + 'P';
            } else {
                sidc += '-P';
            }

            if (n.length > 3) {
                for (let i = 3; i < n.length; i++) {
                    if (n[i].length > 1) {
                        break
                    }
                    sidc += n[i];
                }
            }

            if (sidc.length < 10) {
                sidc += '-'.repeat(10 - sidc.length);
            }

            return sidc.toUpperCase();
        },

        cleanUnit: function (u) {
            let res = {};

            for (const k in u) {
                if (k !== 'marker') {
                    res[k] = u[k];
                }
            }
            return res;
        },

        deleteCurrentUnit: function () {
            if (!this.current_unit_uid) return;
            fetch("unit/" + this.current_unit_uid, {method: "DELETE"})
                .then(function (response) {
                    return response.json()
                })
                .then(this.processUnits);
            // this.removeUnit(this.current_unit_uid);
        },

        sendMessage: function () {
            let msg = {
                from: this.config.callsign,
                from_uid: this.config.uid,
                chatroom: this.chatroom,
                to_uid: this.chat_uid,
                text: this.chat_msg,
            };
            this.chat_msg = "";

            const requestOptions = {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify(msg)
            };
            fetch("/message", requestOptions)
                .then(function (response) {
                    return response.json()
                })
        }
    },
});

function popup(item) {
    let v = '<b>' + item.callsign + '</b><br/>';
    if (item.team) v += item.team + ' ' + item.role + '<br/>';
    if (item.speed && item.speed > 0) v += 'Speed: ' + item.speed.toFixed(0) + ' m/s<br/>';
    if (item.sidc.charAt(2) === 'A') {
        v += "hae: " + item.hae.toFixed(0) + " m<br/>";
    }
    v += item.text.replaceAll('\n', '<br/>').replaceAll('; ', '<br/>');
    return v;
}
