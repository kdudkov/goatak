const app = Vue.createApp({
    data: function () {
        return {
            data: [],
            current: null,
            form: {},
            error: null,
            ts: 0,
        }
    },

    mounted() {
        this.renew();
    },
    computed: {
        all: function () {
            return this.ts && this.data;
        },
    },
    methods: {
        renew: function () {
            let vm = this;

            fetch('/api/device')
                .then(resp => resp.json())
                .then(data => {
                    vm.data = data.sort((a, b) => a.scope.localeCompare(b.scope) || a.login.toLowerCase().localeCompare(b.login.toLowerCase()));
                    vm.ts += 1;
                });
        },
        setCurrent: function (d) {
            this.current = d;
            this.form = {
                callsign: d.callsign,
                role: d.role,
                team: d.team,
                scope: d.scope,
                read_scope: d.read_scope,
                password: '',
            };
        },
        send: function () {
            const requestOptions = {
                method: "PUT",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify(this.form)
            };
            let vm = this;
            fetch('/api/device/' + this.current.login, requestOptions)
                .then(resp => {
                    if (resp.status > 299) {
                        vm.error = 'error ' + resp.status;
                        return null;
                    }
                    return resp.json();
                })
                .then(data => {
                    if (!data) return;

                    if (data.error) {
                        vm.error = data.error;
                        return;
                    }

                    vm.error = "";
                    vm.renew();
                })
                .catch(err => {
                    console.log(err);
                    this.error = err;
                });
        },
        printCoords: function (lat, lng) {
            return lat.toFixed(6) + "," + lng.toFixed(6);
        },
        dt: function (str) {
            if (!str) return "";
            let d = new Date(Date.parse(str));
            return ("0" + d.getDate()).slice(-2) + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" +
                d.getFullYear() + " " + ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
        }
    },
});

app.mount('#app');
