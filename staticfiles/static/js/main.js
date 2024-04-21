
let app = new Vue({
    el: '#app',
    data: {
        units: new Map(),
        connections: new Map(),
        alert: null,
        ts: 0,
    },

    mounted() {
        this.renew();
        this.timer = setInterval(this.renew, 3000);
    },
    computed: {
        all_conns: function () {
            return this.ts && this.connections.values();
        },
    },
    methods: {
        renew: function () {
            let vm = this;
            let units = vm.units;
            let conns = vm.connections;

            fetch('/unit')
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    units.clear();
                    data.forEach(function (i) {
                        units.set(i.uid, i);
                    });
                    vm.ts += 1;
                });
            fetch('/connections')
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    conns.clear();
                    data.forEach(function (i) {
                        conns.set(i.addr, i);
                    });
                    vm.ts += 1;
                });
        },
        byCategory: function (s) {
            let arr = Array.from(this.units.values()).filter(function (u) {
                return u.category === s
            });
            arr.sort(function (a, b) {
                return a.callsign.toLowerCase().localeCompare(b.callsign.toLowerCase());
            });
            return this.ts && arr;
        },
        removeUnit: function (uid) {
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
            return getIconUri(item, false).uri;
        },
        milImg: function (item) {
            return getMilIcon(item, false).uri;
        },
        printCoords: function (lat, lng) {
            return lat.toFixed(6) + "," + lng.toFixed(6);
        },
        dt: function (str) {
            let d = new Date(Date.parse(str));
            return ("0" + d.getDate()).slice(-2) + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" +
                d.getFullYear() + " " + ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
        },
        sp: function (v) {
            return (v * 3.6).toFixed(1);
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