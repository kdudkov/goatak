const app = Vue.createApp({
    data: function () {
        return {
            units: [],
            connections: [],
            ts: 0,
        }
    },

    mounted() {
        this.getData();
        setInterval(this.getData, 3000);
    },
    computed: {
        all_conns: function () {
            let arr = Array.from(this.connections);
            arr.sort(function (a, b) {
                return a.scope.localeCompare(b.scope) || a.user.localeCompare(b.user);
            });
            return this.ts && arr;
        },
    },
    methods: {
        getData: function () {
            let vm = this;

            fetch('/api/unit')
                .then(resp => {
                    if (resp.ok) {
                        return resp.json();
                    }
                    window.location.reload();
                })
                .then(data => {
                    vm.units = data;
                    vm.ts += 1;
                });

            fetch('/api/connections')
                .then(resp => {
                    if (resp.ok) {
                        return resp.json();
                    }
                    window.location.reload();
                })
                .then(data => {
                    vm.connections = data;
                    vm.ts += 1;
                });
        },
        byCategory: function (s) {
            let arr = this.units.filter(function (u) {
                return u.category === s
            });
            arr.sort(function (a, b) {
                return (b.status || '').localeCompare(a.status || '') || a.callsign.toLowerCase().localeCompare(b.callsign.toLowerCase());
            });
            return this.ts && arr;
        },
        getImg: function (item, size) {
            return getIconUri(item, size, false).uri;
        },
        milImg: function (item) {
            return getMilIconUri(item, 24, false).uri;
        },
        sp: function (v) {
            return (v * 3.6).toFixed(1);
        },
        contactsNum: function () {
            let total = this.units.filter(u => u.category === "contact").length;
            let online = this.units.filter(u => u.category === "contact" && u.status === "Online").length;

            return online + "/" + total;
        },
        dt: dtShort,
        printCoords: printCoords,
    },
});

app.mount('#app');
