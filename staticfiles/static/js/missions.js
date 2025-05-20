const app = Vue.createApp({
    data: function () {
        return {
            missions: [],
            current: null,
            alert: null,
            ts: 0,
        }
    },
    mounted() {
        this.renew();
        setInterval(this.renew, 60000);
    },
    methods: {
        renew: function () {
            let vm = this;

            fetch('/api/mission', {redirect: 'manual'})
                .then(resp => {
                    if (!resp.ok) {
                        window.location.reload();
                    }
                    return resp.json();
                })
                .then(data => {
                    vm.missions = data;
                    vm.ts += 1;
                });
        },
        printCoords: printCoords,
        dt: dtShort,
    },
});

app.mount('#app');
