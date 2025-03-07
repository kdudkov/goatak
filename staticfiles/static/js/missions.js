let app = Vue.createApp({
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
    computed: {
        all_missions: function () {
            return this.ts && this.missions;
        },
    },
    methods: {
        renew: function () {
            let vm = this;

            fetch('/api/mission')
                .then(resp => resp.json())
                .then(data => {
                    vm.missions = data;
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
});

app.mount('#app');
