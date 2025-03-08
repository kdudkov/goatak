var map = null;

const app = Vue.createApp({
    data: function () {
        return {
            layers: null,
            conn: null,
            status: "",
            unitsMap: Vue.shallowRef(new Map()),
            messages: [],
            seenMessages: new Set(),
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
        }
    },

    mounted() {
        map = L.map('map');
        map.setView([60, 30], 11);

        L.control.scale({metric: true}).addTo(map);

        this.getConfig();

        let supportsWebSockets = 'WebSocket' in window || 'MozWebSocket' in window;

        if (supportsWebSockets) {
            this.connect();
            // setInterval(this.fetchAllUnits, 60000);
        }

        this.renew();
        setInterval(this.renew, 5000);
        setInterval(this.sender, 1000);

        map.on('click', this.mapClick);
        map.on('mousemove', this.mouseMove);

        this.formFromUnit(null);
    },

    computed: {
        current_unit: function () {
            return this.current_unit_uid ? this.current_unit_uid && this.getCurrentUnit() : null;
        },
        units: function () {
            return this.unitsMap?.value || new Map();
        }
    },

    methods: {
        getConfig: function () {
            let vm = this;

            fetch('/api/config')
                .then(resp => resp.json())
                .then(data => {
                    vm.config = data;

                    map.setView([data.lat, data.lon], data.zoom);

                    if (vm.config.callsign) {
                        vm.me = L.marker([data.lat, data.lon]);
                        vm.me.setIcon(L.icon({
                            iconUrl: "/static/icons/self.png",
                            iconAnchor: new L.Point(16, 16),
                        }));
                        vm.me.addTo(map);

                        fetch('/api/types')
                            .then(resp => resp.json())
                            .then(d => vm.types = d);
                    }

                    layers = L.control.layers({}, null, {hideSingleBase: true});
                    layers.addTo(map);

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
                            l.addTo(map);
                        }
                    });
                });
        },

        connect: function () {
            let url = (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws';
            let vm = this;

            this.fetchAllUnits();
            this.fetchMessages();

            this.conn = new WebSocket(url);

            this.conn.onmessage = function (e) {
                vm.processWS(JSON.parse(e.data));
            };

            this.conn.onopen = function (e) {
                console.log("connected");
                vm.status = "connected";
            };

            this.conn.onerror = function (e) {
                console.log("error");
                vm.status = "error";
            };

            this.conn.onclose = function (e) {
                console.log("closed");
                vm.status = "";
                setTimeout(vm.connect, 3000);
            };
        },

        fetchAllUnits: function () {
            let vm = this;

            fetch('/api/unit')
                .then(resp => resp.json())
                .then(vm.processUnits);
        },

        fetchMessages: function () {
            let vm = this;

            fetch('/api/message')
                .then(resp => resp.json())
                .then(d => vm.messages = d);
        },

        renew: function () {
            if (!this.conn) {
                this.fetchAllUnits();
                this.fetchMessages();
            }
        },

        sender: function () {
            if (this.getTool("dp1")) {
                let p = this.getTool("dp1").getLatLng();

                const requestOptions = {
                    method: "POST",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify({lat: p.lat, lon: p.lng, name: "DP1"})
                };
                fetch("/api/dp", requestOptions);
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

            if (!unit) {
                unit = new Unit(this, u);
                this.units.set(u.uid, unit);
            } else {
                unit.update(u)
            }

            if (this.locked_unit_uid === unit.uid) {
                map.setView(unit.coords());
            }

            return unit;
        },

        processWS: function (u) {
            if (u.type === "unit") {
                this.processUnit(u.unit);
            }

            if (u.type === "delete") {
                this.removeUnit(u.uid);
            }

            if (u.type === "chat") {
                this.fetchMessages();
            }
        },

        removeUnit: function (uid) {
            if (!this.units.has(uid)) return;

            let item = this.units.get(uid);
            item.removeMarker()
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
                return u.unit.category === s
            });
            arr.sort(function (a, b) {
                return a.compare(b);
            });
            return this.ts && arr;
        },

        mapToUnit: function (u) {
            if (u && u.hasCoords()) {
                map.setView(u.coords());
            }
        },

        getImg: function (item, size) {
            return getIconUri(item, size, false).uri;
        },

        milImg: function (item) {
            return getMilIconUri(item, 24, false).uri;
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
                    color: "#ff0000",
                    local: true,
                    send: false,
                }
                if (this.config && this.config.uid) {
                    u.parent_uid = this.config.uid;
                    u.parent_callsign = this.config.callsign;
                }

                let unit = new Unit(null, u);
                unit.post(this);

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
                fetch("/api/pos", requestOptions);
            }
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
                    callsign: u.unit.callsign,
                    category: u.unit.category,
                    type: u.unit.type,
                    subtype: "G",
                    aff: "h",
                    text: u.unit.text,
                    send: u.unit.send,
                    root_sidc: this.types,
                };

                if (u.unit.type.startsWith('a-')) {
                    this.form_unit.type = 'b-m-p-s-m';
                    this.form_unit.aff = u.unit.type.substring(2, 3);
                    this.form_unit.subtype = u.unit.type.substring(4);
                    this.form_unit.root_sidc = this.getRootSidc(u.unit.type.substring(4))
                }
            }
        },

        saveEditForm: function () {
            let u = this.getCurrentUnit();
            if (!u) return;

            u.unit.callsign = this.form_unit.callsign;
            u.unit.category = this.form_unit.category;
            u.unit.send = this.form_unit.send;
            u.unit.text = this.form_unit.text;

            if (this.form_unit.category === "unit") {
                u.unit.type = ["a", this.form_unit.aff, this.form_unit.subtype].join('-');
                u.unit.sidc = this.sidcFromType(u.unit.type);
            } else {
                u.unit.type = this.form_unit.type;
                u.unit.sidc = "";
            }

            u.redraw = true;
            u.updateMarker(this);
            u.post(this);
        },

        getRootSidc: function (s) {
            let curr = this.types;

            for (; ;) {
                if (!curr?.next) {
                    return null;
                }

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

            for (; ;) {
                if (!curr?.next) {
                    return null;
                }

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
                map.removeLayer(p);
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
                let p = new L.marker(coord).addTo(map);
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
                if (u.isContact()) {
                    if (u.isOnline()) online += 1;
                    total += 1;
                }
            })

            return online + "/" + total;
        },

        countByCategory: function (s) {
            let total = 0;
            this.units.forEach(function (u) {
                if (u.unit.category === s) total += 1;
            })

            return total;
        },

        msgNum: function (all) {
            if (!this.messages) return 0;
            let n = 0;
            for (const [key, value] of Object.entries(this.messages)) {
                if (value.messages) {
                    for (m of value.messages) {
                        if (all || !this.seenMessages.has(m.message_id)) n++;
                    }
                }
            }
            return n;
        },

        msgNum1: function (uid, all) {
            if (!this.messages || !this.messages[uid].messages) return 0;
            let n = 0;
            for (m of this.messages[uid].messages) {
                if (all || !this.seenMessages.has(m.message_id)) n++;
            }
            return n;
        },

        openChat: function (uid, chatroom) {
            this.chat_uid = uid;
            this.chatroom = chatroom;
            new bootstrap.Modal(document.getElementById('messages')).show();

            if (this.messages[this.chat_uid]) {
                for (m of this.messages[this.chat_uid].messages) {
                    this.seenMessages.add(m.message_id);
                }
            }
        },

        getStatus: function (uid) {
            return this.ts && this.units.get(uid)?.unit?.status;
        },

        getMessages: function () {
            if (!this.chat_uid) {
                return [];
            }

            let msgs = this.messages[this.chat_uid] ? this.messages[this.chat_uid].messages : [];

            if (document.getElementById('messages').style.display !== 'none') {
                for (m of msgs) {
                    this.seenMessages.add(m.message_id);
                }
            }

            return msgs;
        },

        getUnitName: function (u) {
            let res = u?.unit?.callsign || "no name";
            if (this.config && u.unit.parent_uid === this.config.uid) {
                if (u.unit.send === true) {
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
            if (!s || !s.startsWith('a-')) return "";

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

        deleteCurrentUnit: function () {
            if (!this.current_unit_uid) return;
            fetch("/api/unit/" + this.current_unit_uid, {method: "DELETE"});
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
            let vm = this;
            fetch("/api/message", requestOptions)
                .then(resp => resp.json())
                .then(d => vm.messages = d);

        }
    },
});

app.mount('#app');

class Unit {
    constructor(app, u) {
        this.app = app;
        this.unit = u;
        this.uid = u.uid;

        this.updateMarker();
    }

    update(u) {
        if (this.unit.uid !== u.uid) {
            throw "wrong uid";
        }

        this.redraw = this.needsRedraw(u);

        for (const k of Object.keys(u)) {
            this.unit[k] = u[k];
        }

        this.updateMarker();

        return this;
    }

    needsRedraw(u) {
        if (this.unit.type !== u.type || this.unit.sidc !== u.sidc || this.unit.status !== u.status) return true;
        if (this.unit.speed !== u.speed || this.unit.direction !== u.direction) return true;
        if (this.unit.team !== u.team || this.unit.role !== u.role) return true;

        if (this.unit.sidc.charAt(2) === 'A' && this.unit.hae !== u.hae) return true;
        return false;
    }

    isContact() {
        return this.unit.category === "contact"
    }

    isOnline() {
        return this.unit.status === "Online";
    }

    removeMarker() {
        if (this.marker) {
            map.removeLayer(this.marker);
            this.marker.remove();
            this.marker = null;
        }
    }

    updateMarker() {
        if (!this.hasCoords()) {
            this.removeMarker();
            return;
        }

        if (this.marker) {
            if (this.redraw) {
                this.marker.setIcon(getIcon(this.unit, true));
            }
        } else {
            this.marker = L.marker(this.coords(), {draggable: this.local});
            this.marker.setIcon(getIcon(this.unit, true));

            let vm = this;
            this.marker.on('click', function (e) {
                vm.app.setCurrentUnitUid(vm.uid, false);
            });

            if (this.local) {
                this.marker.on('dragend', function (e) {
                    vm.unit.lat = marker.getLatLng().lat;
                    vm.unit.lon = marker.getLatLng().lng;
                });
            }

            this.marker.addTo(map);
        }

        this.marker.setLatLng(this.coords());
        this.marker.bindTooltip(this.popup());
        this.redraw = false;
    }

    hasCoords() {
        return this.unit.lat && this.unit.lon;
    }

    coords() {
        return [this.unit.lat, this.unit.lon];
    }

    latlng() {
        return L.latLng(this.unit.lat, this.unit.lon)
    }

    compare(u2) {
        return this.unit.callsign.toLowerCase().localeCompare(u2.unit.callsign.toLowerCase());
    }

    popup() {
        let v = '<b>' + this.unit.callsign + '</b><br/>';
        if (this.unit.team) v += this.unit.team + ' ' + this.unit.role + '<br/>';
        if (this.unit.speed) v += 'Speed: ' + this.unit.speed.toFixed(0) + ' m/s<br/>';
        if (this.unit.sidc.charAt(2) === 'A') {
            v += "hae: " + this.unit.hae.toFixed(0) + " m<br/>";
        }
        v += this.unit.text.replaceAll('\n', '<br/>').replaceAll('; ', '<br/>');
        return v;
    }

    post(app) {
        const requestOptions = {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify(this.unit)
        };
        fetch("/api/unit", requestOptions)
            .then(resp => resp.json())
            .then(d => app.processUnit(d));
    }
}
