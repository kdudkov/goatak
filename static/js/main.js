icons = new Map();

function getIcon(name) {
    if (icons.has(name)) {
        return icons.get(name);
    } else {
        icon = L.icon({
            iconUrl: '/static/icons/' + name,
            iconSize: [32, 32],
        });
        icons.set(name, icon);
        return icon;
    }
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
    },

    mounted() {
        this.map = L.map('map').setView([35.462939, -97.537283], 5);
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        }).addTo(this.map);
        L.control.scale({metric: true}).addTo(this.map);

        this.renew();
        this.timer = setInterval(this.renew, 3000);
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
                p.setIcon(getIcon(item.icon))
                // p.bindPopup(popup(item));
                if (this.locked_unit === item.uid) {
                    this.map.setView([item.lat, item.lon]);
                }
                p.on('click', function (e) {
                    app.setUnit(item.uid);
                });
            } else {
                p = L.marker([item.lat, item.lon], {icon: getIcon(item.icon)});
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