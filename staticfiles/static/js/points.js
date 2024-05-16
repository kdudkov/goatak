let app = new Vue({
    el: '#app',
    data: {
        units: new Map(),
        connections: new Map(),
        alert: null,
        ts: 0,
    },

    mounted() {
        this.getData();
        setInterval(this.getData, 3000);
    },
    computed: {
        all_conns: function () {
            let arr = Array.from(this.connections.values());
            arr.sort(function (a, b) {
                return a.scope.localeCompare(b.scope) || a.user.localeCompare(b.user);
            });
            return this.ts && arr;
        },
    },
    methods: {
        getData: function () {
            let vm = this;

            fetch('/unit')
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    vm.units.clear();
                    data.forEach(function (i) {
                        vm.units.set(i.uid, i);
                    });
                    vm.ts += 1;
                });

            fetch('/connections')
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    vm.connections.clear();
                    data.forEach(function (i) {
                        vm.connections.set(i.addr, i);
                    });
                    vm.ts += 1;
                });
        },
        byCategory: function (s) {
            let arr = Array.from(this.units.values()).filter(function (u) {
                return u.category === s
            });
            arr.sort(function (a, b) {
                return (b.status || '').localeCompare(a.status || '') || a.callsign.toLowerCase().localeCompare(b.callsign.toLowerCase());
            });
            return this.ts && arr;
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
    },
});
