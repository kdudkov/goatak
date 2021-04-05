package main

import (
	"errors"
	"net/http"
	"runtime/pprof"

	"github.com/aofei/air"
	"github.com/kdudkov/goatak/model"
)

const (
	baseDir      = "./data"
	infoFileName = "info.json"
)

type HttpServer struct {
	app *App
	air *air.Air
}

func NewHttp(app *App, address string) *HttpServer {
	a := air.New()
	a.Address = address

	a.FILE("/", "static/index.html")
	a.FILE("/map", "static/map.html")
	a.FILES("/static", "static")

	a.GET("/config", getConfigHandler(app))
	a.GET("/units", getUnitsHandler(app))

	a.GET("/stack", getStackHandler())

	addMartiEndpoints(app, a)

	a.NotFoundHandler = getNotFoundHandler(app)

	a.RendererTemplateLeftDelim = "[["
	a.RendererTemplateRightDelim = "]]"

	return &HttpServer{
		app: app,
		air: a,
	}
}

func (h *HttpServer) Serve() error {
	h.app.Logger.Infof("listening http at %s", h.air.Address)
	return h.air.Serve()
}

func getNotFoundHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("404 - %s %s", req.Method, req.Path)
		res.Status = http.StatusNotFound
		return errors.New(http.StatusText(res.Status))
	}
}

func getConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	m := make(map[string]interface{}, 0)
	m["lat"] = app.lat
	m["lon"] = app.lon
	m["zoom"] = app.zoom
	m["version"] = gitRevision
	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON(m)
	}
}

func getUnitsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		units := make([]*model.WebUnit, 0)

		app.units.Range(func(key, value interface{}) bool {
			switch v := value.(type) {
			case *model.Unit:
				units = append(units, v.ToWeb())
			case *model.Contact:
				units = append(units, v.ToWeb())
			}
			return true
		})

		r := make(map[string]interface{}, 0)
		r["units"] = units

		return res.WriteJSON(r)
	}
}

func getStackHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		return pprof.Lookup("goroutine").WriteTo(res.Body, 1)
	}
}
