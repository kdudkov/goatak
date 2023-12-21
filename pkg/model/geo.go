//nolint:gomnd
package model

import (
	"math"
	"sync"
	"time"
)

func DistBea(lat1, lon1, lat2, lon2 float64) (float64, float64) {
	toRadian := math.Pi / 180
	// haversine formula
	// bearing
	y := math.Sin((lon2-lon1)*toRadian) * math.Cos(lat2*toRadian)
	x := math.Cos(lat1*toRadian)*math.Sin(lat2*toRadian) - math.Sin(lat1*toRadian)*math.Cos(lat2*toRadian)*math.Cos((lon2-lon1)*toRadian)
	bea := math.Atan2(y, x) * 180 / math.Pi

	if bea < 0 {
		bea += 360
	}
	// distance
	R := 6371000. // meters
	deltaF := (lat2 - lat1) * toRadian
	deltaL := (lon2 - lon1) * toRadian
	a := math.Sin(deltaF/2)*math.Sin(deltaF/2) + math.Cos(lat1*toRadian)*math.Cos(lat2*toRadian)*math.Sin(deltaL/2)*math.Sin(deltaL/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	dist := R * c

	return dist, bea
}

type Pos struct {
	time  time.Time
	lat   float64
	lon   float64
	speed float64
	ce    float64
	mx    sync.RWMutex
}

func NewPos(lat, lon float64) *Pos {
	return &Pos{lon: lon, lat: lat, speed: 0, ce: 0, time: time.Now(), mx: sync.RWMutex{}}
}

func (p *Pos) Get() (float64, float64) {
	if p == nil {
		return 0, 0
	}

	return p.lat, p.lon
}
