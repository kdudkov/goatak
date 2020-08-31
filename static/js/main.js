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
        points: new Map(),
        map: null,
        ts: 0,
        locked_unit: '',
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
                    let tid = Date.now();
                    let keys = new Set();

                    data.forEach(function (i) {
                        units.set(i.uid, i);
                        vm.updatePoint(i);
                        keys.add(i.uid);
                    });

                    vm.units.forEach(function (v, k) {
                        if (!keys.has(k)) {
                            vm.removePoint(k);
                        }
                    });
                    vm.ts += 1;
                });
        },
        updatePoint: function (item) {
            if (this.points.has(item.uid)) {
                p = this.points.get(item.uid);
                p.setLatLng([item.lat, item.lon], {title: item.callsign});
                p.setIcon(getIcon(item.icon))
                p.bindPopup(popup(item));
                if (this.locked_unit == item.uid) {
                    this.map.flyTo([item.lat, item.lon]);
                }
            } else {
                p = L.marker([item.lat, item.lon], {icon: getIcon(item.icon)});
                this.points.set(item.uid, p);
                p.addTo(this.map);
                p.bindPopup(popup(item));
            }
        },
        removePoint: function (uid) {
            if (this.points.has(uid)) {
                p = this.points.get(uid);
                p.remove();
                this.points.delete(uid);
            }
            this.units.delete(uid);
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