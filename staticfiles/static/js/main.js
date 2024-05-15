
let app = new Vue({
    el: '#app',
    data: {
        connections: new Map(),
        alert: null,
        ts: 0,
    },

    mounted() {
        this.getData();
        setInterval(this.getData, 1000);
    },
    computed: {
        all_conns: function () {
            return this.ts && this.connections.values();
        },
    },
    methods: {
        getData: function () {
            let vm = this;

            fetch('/connections')
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    vm.connections.clear();
                    data.forEach(function (i) {
                        vm.connections.set(i.addr, i);
                    });
                    vm.ts += 1;
                });
        },

    },
});
