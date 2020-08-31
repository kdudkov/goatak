package main

import (
	"goatac/model"
	"runtime/pprof"
	"time"

	"github.com/aofei/air"
)

func NewHttp(app *App, address string) *air.Air {
	srv := air.New()
	srv.Address = address

	srv.FILE("/", "static/index_cl.html")
	srv.FILES("/static", "static")

	srv.GET("/units", getUnitsHandler(app))

	srv.GET("/stack", getStackHandler())

	srv.RendererTemplateLeftDelim = "[["
	srv.RendererTemplateRightDelim = "]]"
	return srv
}

func getUnitsHandler(app *App) func(req *air.Request, res *air.Response) error {

	return func(req *air.Request, res *air.Response) error {
		app.unitsMx.RLock()
		defer app.unitsMx.RUnlock()

		r := make([]*model.WebUnit, 0)

		for _, u := range app.units {
			if u.Stale.After(time.Now()) {
				r = append(r, u.ToWeb())
			}
		}

		return res.WriteJSON(r)
	}
}

func getStackHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		return pprof.Lookup("goroutine").WriteTo(res.Body, 1)
	}
}
