const app = Vue.createApp({
    data: function () {
        return {
            profiles: [],
            current: null,
            form: {},
            error: null,
            ts: 0,
            newOptionKey: '',
            newOptionValue: '',
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
            };
            bootstrap.Modal.getOrCreateInstance(document.getElementById('profile_w')).show();
        },
        send_new: function () {
            let vm = this;

                requestOptions = {
                    method: "POST",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify(this.form)
                };
                url = '/api/profile';

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
        send: function () {
            let vm = this;

                requestOptions = {
                    method: "PUT",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify(this.current)
                };
                url = '/api/profile/' + this.current.login + '/' + this.current.uid;

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
                    vm.renew();
                })
                .catch(err => {
                    console.log(err);
                    this.error = err;
                });
        },
        addOption: function () {
            if (this.newOptionKey && this.newOptionValue) {
                if (!this.current.options) {
                    this.current.options = {};
                }
                this.current.options[this.newOptionKey] = this.newOptionValue;
                this.newOptionKey = '';
                this.newOptionValue = '';
            }
        },
        removeOption: function (key) {
            if (this.current.options && this.current.options.hasOwnProperty(key)) {
                delete this.current.options[key];
            }
        },
        deleteProfile: function () {
            let vm = this;

            if (!this.current) {
                vm.error = 'No profile selected';
                return;
            }

            const requestOptions = {
                method: "DELETE",
                headers: {"Content-Type": "application/json"}
            };
            const url = '/api/profile/' + this.current.login + '/' + this.current.uid;

            fetch(url, requestOptions)
                .then(resp => {
                    if (resp.status > 299) {
                        vm.error = 'Error deleting profile: ' + resp.status;
                        return null;
                    }
                    return resp.json();
                })
                .then(data => {
                    if (data && data.error) {
                        vm.error = data.error;
                        return;
                    }

                    vm.error = "";
                    vm.current = null; // Clear current selection
                    vm.renew(); // Refresh the profiles list
                })
                .catch(err => {
                    console.log(err);
                    vm.error = 'Error deleting profile: ' + err.message;
                });
        },
        printCoords: printCoords,
        dt: dtShort,
    },
});

app.mount('#app');