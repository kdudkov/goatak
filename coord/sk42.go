package coord

import "math"

const ro float64 = 206264.8062 // Число угловых секунд в радиане

// Эллипсоид Красовского
const aP float64 = 6378245          // Большая полуось
const alP float64 = 1 / 298.3       // Сжатие
const e2P float64 = 2*alP - alP*alP // Квадрат эксцентриситета

// Эллипсоид WGS84 (GRS80, эти два эллипсоида сходны по большинству параметров)
const aW float64 = 6378137            // Большая полуось
const alW float64 = 1 / 298.257223563 // Сжатие
const e2W float64 = 2*alW - alW*alW   // Квадрат эксцентриситета

// Вспомогательные значения для преобразования эллипсоидов
const a float64 = (aP + aW) / 2
const e2 float64 = (e2P + e2W) / 2
const da float64 = aW - aP
const de2 float64 = e2W - e2P

// Линейные элементы трансформирования, в метрах
const dx float64 = 23.92
const dy float64 = -141.27
const dz float64 = -80.9

// Угловые элементы трансформирования, в секундах
const wx = 0
const wy = 0
const wz = 0

// Дифференциальное различие масштабов
const ms = 0

func Wgs84_sk42(lat, lon, alt float64) (lat1, lon1 float64) {
	lat1 = lat - dB(lat, lon, alt)/3600
	lon1 = lon - dL(lat, lon, alt)/3600
	return
}

func Sk42_wgs84(lat, lon, alt float64) (lat1, lon1 float64) {
	lat1 = lat + dB(lat, lon, alt)/3600
	lon1 = lon + dL(lat, lon, alt)/3600
	return
}

func dB(Bd, Ld, H float64) float64 {
	b := Bd * math.Pi / 180
	l := Ld * math.Pi / 180

	m := a * (1 - e2) / math.Pow(1-e2*math.Pow(math.Sin(b), 2), 1.5)
	n := a * math.Pow(1-e2*math.Pow(math.Sin(b), 2), -0.5)

	return ro/(m+H)*(n/a*e2*math.Sin(b)*math.Cos(b)*da+(n*n/a/a+1)*n*math.Sin(b)*math.Cos(b)*de2/2-(dx*math.Cos(l)+dy*math.Sin(l))*math.Sin(b)+dz*math.Cos(b)) - wx*math.Sin(l)*(1+e2*math.Cos(2*b)) + wy*math.Cos(l)*(1+e2*math.Cos(2*b)) - ro*ms*e2*math.Sin(b)*math.Cos(b)
}

func dL(Bd, Ld, H float64) float64 {
	b := Bd * math.Pi / 180
	l := Ld * math.Pi / 180

	n := a * math.Pow(1-e2*math.Pow(math.Sin(b), 2), -0.5)
	return ro/((n+H)*math.Cos(b))*(-dx*math.Sin(l)+dy*math.Cos(l)) + math.Tan(b)*(1-e2)*(wx*math.Cos(l)+wy*math.Sin(l)) - wz
}

func WGS84Alt(lat, lon, alt float64) float64 {
	b := lat * math.Pi / 180
	l := lon * math.Pi / 180
	n := a * math.Pow(1-e2*math.Pow(math.Sin(b), 2), -0.5)
	dH := -a/n*da + n*math.Pow(math.Sin(b), 2*de2/2) + (dx*math.Cos(l)+dy*math.Sin(l))*math.Cos(b) + dz*math.Sin(b) - n*e2*math.Sin(b)*math.Cos(b)*(wx/ro*math.Sin(l)-wy/ro*math.Cos(l)) + (a*a/n+alt)*ms
	return alt + dH
}

func Sk42ll2Meters(lat, lon float64) (float64, float64, int) {
	// Номер зоны Гаусса-Крюгера
	zone := (int)(lon/6.0 + 1)

	// Параметры эллипсоида Красовского
	a := 6378245.0                                           // Большая (экваториальная) полуось
	b := 6356863.019                                         // Малая (полярная) полуось
	e2 := (math.Pow(a, 2) - math.Pow(b, 2)) / math.Pow(a, 2) // Эксцентриситет
	n := (a - b) / (a + b)                                   // Приплюснутость

	// Параметры зоны Гаусса-Крюгера
	F := 1.0                                  // Масштабный коэффициент
	Lat0 := 0.0                               // Начальная параллель (в радианах)
	Lon0 := float64(zone*6-3) * math.Pi / 180 // Центральный меридиан (в радианах)
	N0 := 0.0                                 // Условное северное смещение для начальной параллели
	E0 := float64(zone)*1e6 + 500000.0        // Условное восточное смещение для центрального меридиана

	// Перевод широты и долготы в радианы
	latR := lat * math.Pi / 180.0
	lonR := lon * math.Pi / 180.0

	// Вычисление переменных для преобразования
	sinLat := math.Sin(latR)
	cosLat := math.Cos(latR)
	tanLat := math.Tan(latR)

	v := a * F * math.Pow(1-e2*math.Pow(sinLat, 2), -0.5)
	p := a * F * (1 - e2) * math.Pow(1-e2*math.Pow(sinLat, 2), -1.5)
	n2 := v/p - 1
	M1 := (1 + n + 5.0/4.0*math.Pow(n, 2) + 5.0/4.0*math.Pow(n, 3)) * (latR - Lat0)
	M2 := (3*n + 3*math.Pow(n, 2) + 21.0/8.0*math.Pow(n, 3)) * math.Sin(latR-Lat0) * math.Cos(latR+Lat0)
	M3 := (15.0/8.0*math.Pow(n, 2) + 15.0/8.0*math.Pow(n, 3)) * math.Sin(2*(latR-Lat0)) * math.Cos(2*(latR+Lat0))
	M4 := 35.0 / 24.0 * math.Pow(n, 3) * math.Sin(3*(latR-Lat0)) * math.Cos(3*(latR+Lat0))
	M := b * F * (M1 - M2 + M3 - M4)
	I := M + N0
	II := v / 2 * sinLat * cosLat
	III := v / 24 * sinLat * math.Pow(cosLat, 3) * (5 - math.Pow(tanLat, 2) + 9*n2)
	IIIA := v / 720 * sinLat * math.Pow(cosLat, 5) * (61 - 58*math.Pow(tanLat, 2) + math.Pow(tanLat, 4))
	IV := v * cosLat
	V := v / 6 * math.Pow(cosLat, 3) * (v/p - math.Pow(tanLat, 2))
	VI := v / 120 * math.Pow(cosLat, 5) * (5 - 18*math.Pow(tanLat, 2) + math.Pow(tanLat, 4) + 14*n2 - 58*math.Pow(tanLat, 2)*n2)

	// Вычисление северного и восточного смещения (в метрах)
	N := I + II*math.Pow(lonR-Lon0, 2) + III*math.Pow(lonR-Lon0, 4) + IIIA*math.Pow(lonR-Lon0, 6)
	E := E0 + IV*(lonR-Lon0) + V*math.Pow(lonR-Lon0, 3) + VI*math.Pow(lonR-Lon0, 5)

	return N, E, zone
}
