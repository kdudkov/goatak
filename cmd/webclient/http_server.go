package main

import (
	"github.com/aofei/air"
	"runtime/pprof"

	"github.com/kdudkov/goatak/model"
)

func NewHttp(app *App, address string) *air.Air {
	srv := air.New()
	srv.Address = address

	srv.FILE("/", "static/index_cl.html")
	srv.FILES("/static", "static")

	srv.GET("/units", getUnitsHandler(app))
	srv.GET("/config", getConfigHandler(app))

	srv.GET("/stack", getStackHandler())

	srv.RendererTemplateLeftDelim = "[["
	srv.RendererTemplateRightDelim = "]]"
	return srv
}

func getUnitsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.mx.RLock()
		defer app.mx.RUnlock()

		r := make([]*model.WebUnit, 0)

		for _, u := range app.units {
			r = append(r, u.ToWeb())
		}

		return res.WriteJSON(r)
	}
}

func getConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	m := make(map[string]interface{}, 0)
	m["lat"] = app.lat
	m["lon"] = app.lon
	m["zoom"] = app.zoom
	m["myuid"] = app.uid
	m["callsign"] = app.callsign
	m["team"] = app.team
	m["role"] = app.role
	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON(m)
	}
}

func getStackHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		return pprof.Lookup("goroutine").WriteTo(res.Body, 1)
	}
}
