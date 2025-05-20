const app = Vue.createApp({
    data: function () {
        return {
            devices: [],
            login: "",
            current: null,
            form: {},
            scope1: "",
            error: null,
            ts: 0,
        }
    },

    mounted() {
        this.renew();
    },
    methods: {
        renew: function () {
            let vm = this;

            fetch('/api/device', {redirect: 'manual'})
                .then(resp => {
                    if (!resp.ok) {
                        window.location.reload();
                    }
                    return resp.json();
                })
                .then(data => {
                    vm.devices = data.sort((a, b) => a.scope.localeCompare(b.scope) || a.login.toLowerCase().localeCompare(b.login.toLowerCase()));
                    vm.ts += 1;
                });
        },
        create: function () {
            this.current = null;
            this.scope1 = "";
            this.form = {
                callsign: '',
                role: '',
                team: '',
                scope: '',
                read_scope: ['admin', 'public'],
                password: '',
            };
            bootstrap.Modal.getOrCreateInstance(document.getElementById('device_w')).show();
        },
        edit: function () {
            this.scope1 = "";
            this.form = {
                callsign: this.current.callsign,
                role: this.current.role,
                team: this.current.team,
                scope: this.current.scope,
                password: '',
            };

            if (this.current.read_scope) {
                this.form.read_scope = [...this.current.read_scope];
            } else {
                this.form.read_scope = [];
            }

            bootstrap.Modal.getOrCreateInstance(document.getElementById('device_w')).show();
        },
        form_del: function (s) {
            var idx = this.form.read_scope.indexOf(s);
            if (idx !== -1) {
                this.form.read_scope.splice(idx, 1);
            }
        },
        form_add: function () {
            if (!this.scope1) return;
            var idx = this.form.read_scope.indexOf(this.scope1);
            if (idx === -1) {
                this.form.read_scope.push(this.scope1);
            }
            this.scope1 = "";
        },
        send: function () {
            let vm = this;
            let requestOptions = {};
            let url = '';

            if (this.current) {
                requestOptions = {
                    method: "PUT",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify(this.form)
                };
                url = '/api/device/' + this.current.login;
            } else {
                requestOptions = {
                    method: "POST",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify(this.form)
                };
                url = '/api/device';
            }

            fetch(url, requestOptions)
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
                    bootstrap.Modal.getOrCreateInstance(document.getElementById('device_w')).hide();
                    vm.renew();
                })
                .catch(err => {
                    console.log(err);
                    this.error = err;
                });
        },
        printCoords: printCoords,
        dt: dtShort,
    },
});

app.mount('#app');
