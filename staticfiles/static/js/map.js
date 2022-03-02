const colors = new Map([
    ['White', 'white'],
    ['Yellow', 'yellow'],
    ['Orange', 'orange'],
    ['Magenta', 'magenta'],
    ['Red', 'red'],
    ['Maroon', 'maroon'],
    ['Purple', 'purple'],
    ['Dark Blue', 'darkblue'],
    ['Blue', 'blue'],
    ['Cyan', 'cyan'],
    ['Teal', 'teal'],
    ['Green', 'green'],
    ['Dark Green', 'darkgreen'],
    ['Brown', 'brown'],
]);

function ne(s) {
    return s !== undefined && s !== null && s !== "";
}

function getIcon(item, withText) {
    if (item.category === "contact") {
        return {uri: toUri(roleCircle(24, colors.get(item.team), '#000', item.role)), x: 12, y: 12};
    }
    if (ne(item.icon) && item.icon.startsWith("COT_MAPPING_SPOTMAP/")) {
        return {uri: toUri(circle(16, ne(item.color) ? item.color : 'green', '#000', null)), x: 8, y: 8}
    }
    if (item.category === "point") {
        return {uri: toUri(circle(16, ne(item.color) ? item.color : 'green', '#000', null)), x: 8, y: 8}
    }
    return getMilIcon(item, withText);
}

function getMilIcon(item, withText) {
    let opts = {size: 24};
    if (withText) {
        // opts['uniqueDesignation'] = item.callsign;
        if (item.speed > 0) {
            opts['speed'] = (item.speed * 3.6).toFixed(1) + " km/h";
            opts['direction'] = item.course;
        }
    }

    let symb = new ms.Symbol(item.sidc, opts);
    return {uri: symb.toDataURL(), x: symb.getAnchor().x, y: symb.getAnchor().y}
}

let app = new Vue({
    el: '#app',
    data: {
        units: new Map(),
        markers: new Map(),
        map: null,
        ts: 0,
        locked_unit_uid: '',
        current_unit: null,
        config: null,
        tools: new Map(),
        coords: null,
        point_num: 1,
    },

    mounted() {
        this.map = L.map('map');
        let osm = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
            maxZoom: 19
        });
        let topoAttribution = 'Kartendaten: &copy; <a href="https://openstreetmap.org/copyright">OpenStreetMap</a>-Mitwirkende, <a href="http://viewfinderpanoramas.org">SRTM</a> | Kartendarstellung: &copy; <a href="https://opentopomap.org">OpenTopoMap</a>';
        let opentopo = L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
            attribution: topoAttribution,
            maxZoom: 17
        });
        let google = L.tileLayer('http://{s}.google.com/vt/lyrs=y&x={x}&y={y}&z={z}&s=Galileo', {
            subdomains: ['mt1', 'mt2', 'mt3'],
            maxZoom: 20
        });
        let sputnik = L.tileLayer('https://{s}.tilessputnik.ru/{z}/{x}/{y}.png', {
            maxZoom: 20
        });
        osm.addTo(this.map);

        L.control.scale({metric: true}).addTo(this.map);
        L.control.layers({
            "OSM": osm,
            "Telesputnik": sputnik,
            "OpenTopoMap": opentopo,
            "Google sat": google
        }, null, {hideSingleBase: true}).addTo(this.map);

        this.renew();
        this.timer = setInterval(this.renew, 3000);

        this.map.on('click', this.mapClick);
        this.map.on('mousemove', this.mouseMove);

        let vm = this;
        fetch('/config')
            .then(function (response) {
                return response.json()
            })
            .then(function (data) {
                vm.config = data;
                if (vm.map != null) {
                    vm.map.setView([data.lat, data.lon], data.zoom);
                }
            });
    },
    computed: {},
    methods: {
        renew: function () {
            let vm = this;
            let units = vm.units;

            fetch('/units')
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    let keys = new Set();

                    data.units.forEach(function (i) {
                        let oldUnit = units.get(i.uid);
                        units.set(i.uid, i);
                        vm.updateMarker(i, false, oldUnit != null && oldUnit.sidc !== i.sidc);
                        keys.add(i.uid);
                        if (vm.current_unit != null && vm.current_unit.uid === i.uid) {
                            vm.current_unit = i;
                        }
                    });

                    vm.units.forEach(function (v, k) {
                        if (v.my === undefined && !keys.has(k)) {
                            vm.removeUnit(k);
                        }
                    });
                    vm.ts += 1;
                });

            if (this.getTool("dp1") != null) {
                let p = this.getTool("dp1").getLatLng();

                const requestOptions = {
                    method: "POST",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify({lat: p.lat, lon: p.lng, name: "DP1"})
                };
                fetch("/dp", requestOptions);
            }
        },
        updateMarker: function (item, draggable, updateIcon) {
            if (item.lon === 0 && item.lat === 0) {
                return
            }
            let marker;
            if (this.markers.has(item.uid)) {
                marker = this.markers.get(item.uid);
                if (updateIcon) {
                    let icon = getIcon(item, true);
                    marker.setIcon(L.icon({
                        iconUrl: icon.uri,
                        iconAnchor: new L.Point(icon.x, icon.y),
                    }));
                }
            } else {
                marker = L.marker([item.lat, item.lon], {draggable: draggable});
                this.markers.set(item.uid, marker);
                marker.on('click', function (e) {
                    app.setCurrentUnit(item.uid);
                });
                if (draggable) {
                    marker.on('dragend', function (e) {
                        item.lat = marker.getLatLng().lat;
                        item.lon = marker.getLatLng().lng;
                    });
                }
                let icon = getIcon(item, true);
                marker.setIcon(L.icon({
                    iconUrl: icon.uri,
                    iconAnchor: new L.Point(icon.x, icon.y),
                }));
                marker.addTo(this.map);
            }
            marker.setLatLng([item.lat, item.lon]);
            marker.bindTooltip(popup(item));
            if (this.locked_unit_uid === item.uid) {
                this.map.setView([item.lat, item.lon]);
            }
        },
        removeUnit: function (uid) {
            if (this.markers.has(uid)) {
                p = this.markers.get(uid);
                this.map.removeLayer(p);
                p.remove();
                this.markers.delete(uid);
            }
            this.units.delete(uid);
            if (this.current_unit != null && this.current_unit.uid === uid) {
                this.current_unit = null;
            }
        },
        setCurrentUnit: function (uid) {
            if (this.units.has(uid)) {
                this.current_unit = this.units.get(uid);
                this.mapToUnit(this.unit);
            }
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
            if (u == null) {
                return;
            }
            if (u.lat !== 0 || u.lon !== 0) {
                this.map.setView([u.lat, u.lon]);
            }
        },
        getImg: function (item) {
            return getIcon(item, false).uri;
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
        mapClick: function (e) {
            if (document.getElementById("redx").checked === true) {
                this.addOrMove("redx", e.latlng, "/static/icons/x.png")
                return;
            }
            if (document.getElementById("dp1").checked === true) {
                this.addOrMove("dp1", e.latlng, "/static/icons/spoi_icon.png")
                return;
            }
            if (document.getElementById("point").checked === true) {
                let uid = uuidv4();
                let u = {
                    uid: uid,
                    category: "point",
                    callsign: "point-" + this.point_num++,
                    stale: null,
                    received: null,
                    type: "b-m-p-s-m",
                    lat: e.latlng.lat,
                    lon: e.latlng.lng,
                    hae: 0,
                    speed: 0,
                    course: 0,
                    status: "",
                    text: "",
                    my: true,
                }
                console.log(u);
                this.units.set(uid, u);
                this.updateMarker(u, true, false);
                this.current_unit = u;
            }
        },
        mouseMove: function (e) {
            this.coords = e.latlng;
        },
        removeTool: function (name) {
            if (this.tools.has(name)) {
                p = this.tools.get(name);
                this.map.removeLayer(p);
                p.remove();
                this.tools.delete(name);
                this.ts++;
            }
        },
        getTool: function (name) {
            if (this.ts > 5) {
            }
            return this.tools.get(name);
        },
        addOrMove(name, coord, icon) {
            if (this.tools.has(name)) {
                this.tools.get(name).setLatLng(coord);
            } else {
                let p = new L.marker(coord).addTo(this.map);
                if (ne(icon)) {
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
            if (this.messages == null) return 0;
            return this.messages.length;
        },
        ne: function (s) {
            return s !== undefined && s !== null && s !== "";
        }
    },
});

function popup(item) {
    let v = '<b>' + item.callsign + '</b><br/>';
    if (ne(item.team)) v += item.team + ' ' + item.role + '<br/>';
    if (ne(item.speed)) v += 'Speed: ' + item.speed.toFixed(0) + '<br/>';
    v += item.text;
    return v;
}

function circle(size, color, bg, text) {
    let x = Math.round(size / 2);
    let r = x - 1;

    let s = '<svg width="' + size + '" height="' + size + '" xmlns="http://www.w3.org/2000/svg"><metadata id="metadata1">image/svg+xml</metadata>';
    s += '<circle style="fill: ' + color + '; stroke: ' + bg + ';" cx="' + x + '" cy="' + x + '" r="' + r + '"/>';

    if (text != null && text !== '') {
        s += '<text x="50%" y="50%" text-anchor="middle" font-size="12px" font-family="Arial" dy=".3em">' + text + '</text>';
    }
    s += '</svg>';
    return s;
}

function roleCircle(size, color, bg, role) {
    let t = '';
    if (role === 'HQ') {
        t = 'HQ';
    } else if (role === 'Team Lead') {
        t = 'TL';
    } else if (role === 'K9') {
        t = 'K9';
    } else if (role === 'Forward Observer') {
        t = 'FO';
    } else if (role === 'Sniper') {
        t = 'S';
    } else if (role === 'Medic') {
        t = 'M';
    } else if (role === 'RTO') {
        t = 'R';
    }

    return circle(size, color, bg, t);
}

function toUri(s) {
    return encodeURI("data:image/svg+xml," + s).replaceAll("#", "%23");
}

function uuidv4() {
    return ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, c =>
        (c ^ crypto.getRandomValues(new Uint8Array(1))[0] & 15 >> c / 4).toString(16)
    );
}
