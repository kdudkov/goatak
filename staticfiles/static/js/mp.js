let app = Vue.createApp({
    data: function () {
        return {
            mp: [],
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
            return this.ts && this.mp;
        },
    },
    methods: {
        renew: function () {
            let vm = this;

            fetch('/mp')
                .then(resp => resp.json())
                .then(function (data) {
                    vm.mp = data.sort((a, b) => a.Scope.localeCompare(b.Scope) || a.Name.toLowerCase().localeCompare(b.Name.toLowerCase()));
                    vm.ts += 1;
                });
        },
        printCoords: function (lat, lng) {
            return lat.toFixed(6) + "," + lng.toFixed(6);
        },
        dt: function (str) {
            let d = new Date(Date.parse(str));
            return ("0" + d.getDate()).slice(-2) + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" +
                d.getFullYear() + " " + ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
        }
    },
}).mount('#app');