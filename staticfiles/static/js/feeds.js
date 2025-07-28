const app = Vue.createApp({
    data: function () {
        return {
            feeds: [],
            current: null,
            form: {},
            error: null,
            ts: 0,
            hls: null,
        }
    },

    mounted() {
        this.renew();
        this.hls = new Hls();
    },
    methods: {
        renew: function () {
            let vm = this;

            fetch('/api/feed', {redirect: 'manual'})
                .then(resp => {
                    if (!resp.ok) {
                        window.location.reload();
                    }
                    return resp.json();
                })
                .then(data => {
                    vm.feeds = data.sort((a, b) => a.uid.toLowerCase().localeCompare(b.uid.toLowerCase()));
                    vm.ts += 1;
                });
        },
        setCurrent: function (f) {
            this.current = f;
            let video = document.getElementById('video');

            if (!video) return;

            if (f.url.startsWith('http')) {
                if (video.canPlayType('application/vnd.apple.mpegurl')) {
                    video.src = f.url;
                    video.addEventListener('canplay', () => video.play());

                    return
                }
            }

            if (Hls.isSupported()) {
                this.hls.attachMedia(video);
                this.hls.loadSource(f.url);
                this.hls.on(Hls.Events.MANIFEST_PARSED, () => video.play());
            }
        },
        create: function () {
            this.current = null;
            this.form = {
                uid: '',
                alias: '',
                active: true,
                url: '',
                scope: '',
                lat: 0,
                lon: 0,
                fov: '',
                heading: '',
                range: '',
            };
            bootstrap.Modal.getOrCreateInstance(document.getElementById('feed_w')).show();
        },
        send_new: function () {
            let vm = this;

            requestOptions = {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify(this.form)
            };
            url = '/api/feed';

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
                    bootstrap.Modal.getOrCreateInstance(document.getElementById('feed_w')).hide();
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
            url = '/api/feed/' + this.current.uid;

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
        deleteFeed: function () {
            let vm = this;

            if (!this.current) {
                vm.error = 'No feed selected';
                return;
            }

            const requestOptions = {
                method: "DELETE",
                headers: {"Content-Type": "application/json"}
            };
            const url = '/api/feed/' + this.current.uid;

            fetch(url, requestOptions)
                .then(resp => {
                    if (resp.status > 299) {
                        vm.error = 'Error deleting feed: ' + resp.status;
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
                    vm.renew(); // Refresh the feeds list
                })
                .catch(err => {
                    console.log(err);
                    vm.error = 'Error deleting feed: ' + err.message;
                });
        },
        printCoords: printCoords,
        dt: dtShort,
    },
});

app.mount('#app');
