const app = Vue.createApp({
    data: function () {
        return {
            data: [],
            current: null,
            alert: null,
            ts: 0,
        }
    },

    mounted() {
        this.renew();
        setInterval(this.renew, 60000);
    },
    computed: {
        all: function () {
            return this.ts && this.data;
        },
    },
    methods: {
        renew: function () {
            let vm = this;

            fetch('/api/point')
                .then(resp => {
                    if (resp.ok) {
                        return resp.json();
                    }
                    window.location.reload();
                })
                .then(function (data) {
                    vm.data = data.sort((a, b) => a.Scope.localeCompare(b.Scope) || a.Callsign.toLowerCase().localeCompare(b.Callsign.toLowerCase()));
                    vm.ts += 1;
                });
        },
        printCoords: printCoords,
        dt: dtShort,
    },
});

app.mount('#app');
