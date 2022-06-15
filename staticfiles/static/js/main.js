const colors = new Map([
    ['White', 'white'],
    ['Yellow', 'yellow'],
    ['Orange', 'orange'],
    ['Magenta', 'magenta'],
    ['Red', 'red'],
    ['Maroon', 'maroon'],
    ['Purple', 'purple'],
    ['Dark Blue', 'darkblue'],
    ['Blue', 'blue'],
    ['Cyan', 'cyan'],
    ['Teal', 'teal'],
    ['Green', 'green'],
    ['Dark Green', 'darkgreen'],
    ['Brown', 'brown'],
]);

function getIcon(item, withText) {
    if (item.team !== "") {
        return {uri: toUri(roleCircle(24, colors.get(item.team), '#000', item.role)), x: 12, y: 12};
    }
    if (item.icon !== undefined && item.icon.startsWith("COT_MAPPING_SPOTMAP/")) {
        return {uri: toUri(circle(16, item.color === '' ? 'green' : item.color, '#000', null)), x: 5, y: 5}
    }
    if (item.icon !== undefined) {
        return {uri: toUri(circle(16, item.color === '' ? 'green' : item.color, '#000', null)), x: 5, y: 5}
    }
    return getMilIcon(24, item, withText);
}

function getMilIcon(size, item, withText) {
    let opts = {size: size};
    if (withText) {
        opts['uniqueDesignation'] = item.callsign;
    }
    if (withText && item.speed > 0) {
        opts['speed'] = (item.speed * 3.6).toFixed(1) + " km/h";
        opts['direction'] = item.course;
    }

    let symb = new ms.Symbol(item.sidc, opts);
    return {uri: symb.toDataURL(), x: symb.getAnchor().x, y: symb.getAnchor().y}
}

let app = new Vue({
    el: '#app',
    data: {
        units: new Map(),
        connections: new Map(),
        alert: null,
        ts: 0,
    },

    mounted() {
        this.renew();
        this.timer = setInterval(this.renew, 3000);
    },
    computed: {
        all_conns: function () {
            return this.ts && this.connections.values();
        },
    },
    methods: {
        renew: function () {
            let vm = this;
            let units = vm.units;
            let conns = vm.connections;

            fetch('/unit')
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    units.clear();
                    data.units.forEach(function (i) {
                        units.set(i.uid, i);
                    });
                    vm.ts += 1;
                });
            fetch('/connections')
                .then(function (response) {
                    return response.json()
                })
                .then(function (data) {
                    conns.clear();
                    data.forEach(function (i) {
                        conns.set(i.uid, i);
                    });
                    vm.ts += 1;
                });
        },
        byCategory: function (s) {
            let arr = Array.from(this.units.values()).filter(function (u) {
                return u.category === s
            });
            arr.sort(function (a, b) {
                let ua = a.callsign.toLowerCase(), ub = b.callsign.toLowerCase();
                if (ua < ub) return -1;
                if (ua > ub) return 1;
                return 0;
            });
            return this.ts && arr;
        },
        removeUnit: function (uid) {
            this.units.delete(uid);
            if (this.unit != null && this.unit.uid === uid) {
                this.unit = null;
            }
        },
        setUnit: function (uid) {
            if (this.units.has(uid)) {
                this.unit = this.units.get(uid);
            }
        },
        getImg: function (item) {
            return getIcon(item, false).uri;
        },
        milImg: function (item) {
            return getMilIcon(item, false).uri;
        },
        printCoords: function (lat, lng) {
            return lat.toFixed(6) + "," + lng.toFixed(6);
        },
        dt: function (str) {
            let d = new Date(Date.parse(str));
            return ("0" + d.getDate()).slice(-2) + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" +
                d.getFullYear() + " " + ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
        },
        sp: function (v) {
            return (v * 3.6).toFixed(1);
        }
    },
});

function popup(item) {
    let v = '<h5>' + item.callsign + '</h5>';
    v += 'Team: ' + item.team + '<br/>';
    v += 'Role: ' + item.role + '<br/>';
    v += 'Speed: ' + item.speed + '<br/>';
    v += item.text;
    return v;
}

function circle(size, color, bg, text) {
    let x = Math.round(size / 2);
    let r = x - 1;

    let s = '<svg width="' + size + '" height="' + size + '" xmlns="http://www.w3.org/2000/svg"><metadata id="metadata1">image/svg+xml</metadata>';
    s += '<circle style="fill: ' + color + '; stroke: ' + bg + ';" cx="' + x + '" cy="' + x + '" r="' + r + '"/>';

    if (text != null && text !== '') {
        s += '<text x="50%" y="50%" text-anchor="middle" font-size="12px" font-family="Arial" dy=".3em">' + text + '</text>';
    }
    s += '</svg>';
    return s;
}

function roleCircle(size, color, bg, role) {
    let t = '';
    if (role === 'HQ') {
        t = 'HQ';
    } else if (role === 'Team Lead') {
        t = 'TL';
    } else if (role === 'K9') {
        t = 'K9';
    } else if (role === 'Forward Observer') {
        t = 'FO';
    } else if (role === 'Sniper') {
        t = 'S';
    } else if (role === 'Medic') {
        t = 'M';
    } else if (role === 'RTO') {
        t = 'R';
    }

    return circle(size, color, bg, t);
}

function toUri(s) {
    return encodeURI("data:image/svg+xml," + s).replaceAll("#", "%23");
}