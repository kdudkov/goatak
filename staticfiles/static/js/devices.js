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
                .then(function (data) {
                    vm.data = data.sort((a, b) => a.scope.localeCompare(b.scope) || a.login.toLowerCase().localeCompare(b.login.toLowerCase()));
                    vm.ts += 1;
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
