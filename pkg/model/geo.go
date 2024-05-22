//nolint:gomnd
package model

import (
	"math"
	"time"

	"github.com/kdudkov/goatak/pkg/cot"
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
	Time  time.Time
	Lat   float64
	Lon   float64
	Alt   float64
	Speed float64
	Track float64
	Ce    float64
}

func NewPos(lat, lon float64) *Pos {
	return NewPosFull(lat, lon, 0, 0, 0)
}

func NewPosFull(lat, lon, alt, speed, track float64) *Pos {
	return &Pos{Lon: lon, Lat: lat, Alt: alt, Speed: speed, Track: track, Ce: 0, Time: time.Now()}
}

func msg2pos(msg *cot.CotMessage) *Pos {
	return &Pos{
		Time:  msg.GetSendTime(),
		Lat:   msg.GetLat(),
		Lon:   msg.GetLon(),
		Alt:   msg.GetTakMessage().GetCotEvent().GetHae(),
		Ce:    msg.GetTakMessage().GetCotEvent().GetCe(),
		Speed: msg.GetTakMessage().GetCotEvent().GetDetail().GetTrack().GetSpeed(),
		Track: msg.GetTakMessage().GetCotEvent().GetDetail().GetTrack().GetCourse(),
	}
}

func (p *Pos) GetCoord() (float64, float64) {
	if p == nil {
		return 0, 0
	}

	return p.Lat, p.Lon
}

func (p *Pos) GetLat() float64 {
	if p == nil {
		return 0
	}

	return p.Lat
}

func (p *Pos) GetLon() float64 {
	if p == nil {
		return 0
	}

	return p.Lon
}

func (p *Pos) GetAlt() float64 {
	if p == nil {
		return 0
	}

	return p.Alt
}

func (p *Pos) GetSpeed() float64 {
	if p == nil {
		return 0
	}

	return p.Speed
}

func (p *Pos) GetTrack() float64 {
	if p == nil {
		return 0
	}

	return p.Track
}

func (p *Pos) GetCe() float64 {
	if p == nil {
		return cot.NotNum
	}

	return p.Ce
}
