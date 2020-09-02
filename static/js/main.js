function getIcon(item) {
    if (item.icon != "") {
        icon = L.icon({
            iconUrl: '/static/icons/' + item.icon,
            iconSize: [24, 24],
            iconAnchor: [12, 12]
        });
        return icon;
    }
    return milIcon(item);
}

function milIcon(item) {
    opts = {uniqueDesignation: item.callsign, size: 24};
    if (item.speed > 0) {
        opts['speed'] = item.speed.toFixed(1) + " m/s";
        opts['direction'] = item.course;
    }

    let symb = new ms.Symbol(item.sidc, opts);

    return L.icon({
        iconUrl: symb.toDataURL(),
        iconAnchor: new L.Point(symb.getAnchor().x, symb.getAnchor().y)
    });
}

let app = new Vue({
    el: '#app',
    data: {
        units: new Map(),
        markers: new Map(),
        map: null,
        ts: 0,
        locked_unit: '',
        unit: null,
        config: null,
    },

    mounted() {
        this.map = L.map('map');
        let osm = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        });
        let topoAttribution = 'Kartendaten: &copy; <a href="https://openstreetmap.org/copyright">OpenStreetMap</a>-Mitwirkende, <a href="http://viewfinderpanoramas.org">SRTM</a> | Kartendarstellung: &copy; <a href="https://opentopomap.org">OpenTopoMap</a>';
        let opentopo = L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
            attribution: topoAttribution
        });
        let google = L.tileLayer('http://{s}.google.com/vt/lyrs=y&x={x}&y={y}&z={z}&s=Galileo', {
            subdomains: ['mt1', 'mt2', 'mt3']
        });
        osm.addTo(this.map);

        L.control.scale({metric: true}).addTo(this.map);
        L.control.layers({
            "OSM": osm,
            "OpenTopoMap": opentopo,
            "Google sat": google
        }, null, {hideSingleBase: true}).addTo(this.map);

        this.renew();
        this.timer = setInterval(this.renew, 3000);

        let vm = this;
        fetch('/config')
            .then(function (response) {
                return response.json()
            })
            .then(function (data) {
                vm.config = data;
                vm.map.setView([data.lat, data.lon], data.zoom);
            });
    },
    computed: {
        all_units: function () {
            let arr = Array.from(this.units.values());
            arr.sort(function (a, b) {
                var ua = a.callsign.toLowerCase(), ub = b.callsign.toLowerCase();
                if (ua < ub) return -1;
                if (ua > ub) return 1;
                return 0;
            });
            return this.ts && arr;
        }
    },

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

                    data.forEach(function (i) {
                        units.set(i.uid, i);
                        vm.updateMarker(i);
                        keys.add(i.uid);
                        if (vm.unit != null && vm.unit.uid === i.uid) {
                            vm.unit = i;
                        }
                    });

                    vm.units.forEach(function (v, k) {
                        if (!keys.has(k)) {
                            vm.removeUnit(k);
                        }
                    });
                    vm.ts += 1;
                });
        },
        updateMarker: function (item) {
            if (this.markers.has(item.uid)) {
                p = this.markers.get(item.uid);
                p.setLatLng([item.lat, item.lon], {title: item.callsign});
                p.setIcon(getIcon(item))
                // p.bindPopup(popup(item));
                if (this.locked_unit === item.uid) {
                    this.map.setView([item.lat, item.lon]);
                }
                p.on('click', function (e) {
                    app.setUnit(item.uid);
                });
            } else {
                p = L.marker([item.lat, item.lon], {icon: getIcon(item)});
                this.markers.set(item.uid, p);
                p.addTo(this.map);
                // p.bindPopup(popup(item));
                p.on('click', function (e) {
                    app.setUnit(item.uid);
                });
            }
        },
        removeUnit: function (uid) {
            if (this.markers.has(uid)) {
                p = this.markers.get(uid);
                p.remove();
                this.markers.delete(uid);
            }
            this.units.delete(uid);
            if (this.unit != null && this.unit.uid === uid) {
                this.unit = null;
            }
        },
        setUnit: function (uid) {
            if (this.units.has(uid)) {
                this.unit = this.units.get(uid);
            }
        },
        getImg: function (item) {
            if (item.icon != "") {
                return '/static/icons/' + item.icon;
            }
            return new ms.Symbol(item.sidc, {size: 24}).toDataURL();
        },
        milImg: function (item) {
            return new ms.Symbol(item.sidc, {size: 24}).toDataURL();
        },
        dt: function (str) {
            let d = new Date(Date.parse(str));
            return ("0" + d.getDate()).slice(-2) + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" +
                d.getFullYear() + " " + ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
        }
    },
});

function popup(item) {
    let v = '<h5>' + item.callsign + '</h5>';
    v += 'Team: ' + item.team + '<br/>';
    v += 'Role: ' + item.role + '<br/>';
    v += 'Speed: ' + item.speed + '<br/>';
    v += item.text;
    return v;
}