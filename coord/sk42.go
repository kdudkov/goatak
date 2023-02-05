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

func WGS84_SK42(lat, lon, alt float64) (lat1, lon1 float64) {
	lat1 = lat - dB(lat, lon, alt)/3600
	lon1 = lon - dL(lat, lon, alt)/3600
	return
}

func SK42_WGS84(lat, lon, alt float64) (lat1, lon1 float64) {
	lat1 = lat + dB(lat, lon, alt)/3600
	lon1 = lon + dL(lat, lon, alt)/3600
	return
}

func dB(Bd, Ld, H float64) float64 {
	B := Bd * math.Pi / 180
	L := Ld * math.Pi / 180

	M := a * (1 - e2) / math.Pow(1-e2*math.Pow(math.Sin(B), 2), 1.5)
	N := a * math.Pow(1-e2*math.Pow(math.Sin(B), 2), -0.5)

	return ro/(M+H)*(N/a*e2*math.Sin(B)*math.Cos(B)*da+(N*N/a/a+1)*N*math.Sin(B)*math.Cos(B)*de2/2-(dx*math.Cos(L)+dy*math.Sin(L))*math.Sin(B)+dz*math.Cos(B)) - wx*math.Sin(L)*(1+e2*math.Cos(2*B)) + wy*math.Cos(L)*(1+e2*math.Cos(2*B)) - ro*ms*e2*math.Sin(B)*math.Cos(B)
}

func dL(Bd, Ld, H float64) float64 {
	B := Bd * math.Pi / 180
	L := Ld * math.Pi / 180

	N := a * math.Pow(1-e2*math.Pow(math.Sin(B), 2), -0.5)
	return ro/((N+H)*math.Cos(B))*(-dx*math.Sin(L)+dy*math.Cos(L)) + math.Tan(B)*(1-e2)*(wx*math.Cos(L)+wy*math.Sin(L)) - wz
}

func WGS84Alt(lat, lon, alt float64) float64 {
	B := lat * math.Pi / 180
	L := lon * math.Pi / 180
	N := a * math.Pow(1-e2*math.Pow(math.Sin(B), 2), -0.5)
	dH := -a/N*da + N*math.Pow(math.Sin(B), 2*de2/2) + (dx*math.Cos(L)+dy*math.Sin(L))*math.Cos(B) + dz*math.Sin(B) - N*e2*math.Sin(B)*math.Cos(B)*(wx/ro*math.Sin(L)-wy/ro*math.Cos(L)) + (a*a/N+alt)*ms
	return alt + dH
}
