const app = Vue.createApp({
    data: function () {
        return {
            profiles: [],
            current: null,
            form: {},
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

            fetch('/api/profile', {redirect: 'manual'})
                .then(resp => {
                    if (!resp.ok) {
                        window.location.reload();
                    }
                    return resp.json();
                })
                .then(data => {
                    vm.profiles = data.sort((a, b) => a.login.toLowerCase().localeCompare(b.login.toLowerCase()));
                    vm.ts += 1;
                });
        },
        create: function () {
            this.current = null;
            this.form = {
                login: '',
                uid: '',
                callsign: '',
                team: '',
                role: '',
                cot_type: '',
                options: {},
            };
            bootstrap.Modal.getOrCreateInstance(document.getElementById('profile_w')).show();
        },
        edit: function () {
            this.form = {
                callsign: this.current.callsign,
                team: this.current.team,
                role: this.current.role,
                cot_type: this.current.cot_type,
                options: this.current.options || {},
            };

            bootstrap.Modal.getOrCreateInstance(document.getElementById('profile_w')).show();
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
                url = '/api/profile/' + this.current.login + '/' + this.current.uid;
            } else {
                requestOptions = {
                    method: "POST",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify(this.form)
                };
                url = '/api/profile';
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
                    bootstrap.Modal.getOrCreateInstance(document.getElementById('profile_w')).hide();
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