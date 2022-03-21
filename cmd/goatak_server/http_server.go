package main

import (
	"embed"
	"errors"
	"net/http"
	"runtime/pprof"

	"github.com/aofei/air"
	"github.com/kdudkov/goatak/model"
	"github.com/kdudkov/goatak/staticfiles"
)

//go:embed templates
var templates embed.FS

type Connection struct {
	Uid  string `json:"uid"`
	User string `json:"user"`
	Ssl  bool   `json:"ssl"`
	Ver  int32  `json:"ver"`
}

type HttpServer struct {
	app      *App
	air      *air.Air
	api      *air.Air
	renderer *staticfiles.Renderer
}

func NewHttp(app *App, address string, apiAddress string) *HttpServer {
	a := air.New()
	a.Address = address

	staticfiles.EmbedFiles(a, "/static")

	renderer := new(staticfiles.Renderer)
	renderer.LeftDelimeter = "[["
	renderer.RightDelimeter = "]]"
	renderer.Load(templates, "templates")

	a.GET("/", getIndexHandler(app, renderer))
	a.GET("/map", getMapHandler(app, renderer))
	a.GET("/config", getConfigHandler(app))
	a.GET("/units", getUnitsHandler(app))
	a.GET("/connections", getConnHandler(app))

	a.GET("/stack", getStackHandler())

	a.RendererTemplateLeftDelim = "[["
	a.RendererTemplateRightDelim = "]]"

	api := air.New()
	api.Address = apiAddress

	//if app.config.keyFile != "" && app.config.certFile != "" {
	//	api.TLSCertFile = app.config.certFile
	//	api.TLSKeyFile = app.config.keyFile
	//}
	addMartiEndpoints(app, api)

	api.NotFoundHandler = getNotFoundHandler(app)

	return &HttpServer{
		app:      app,
		air:      a,
		api:      api,
		renderer: renderer,
	}
}

func (h *HttpServer) Start() {
	go func() {
		h.app.Logger.Infof("listening http at %s", h.air.Address)
		if err := h.air.Serve(); err != nil {
			h.app.Logger.Panicf(err.Error())
		}
	}()
	go func() {
		h.app.Logger.Infof("listening api calls at %s", h.api.Address)
		if err := h.api.Serve(); err != nil {
			h.app.Logger.Panicf(err.Error())
		}
	}()
}

func getIndexHandler(app *App, r *staticfiles.Renderer) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		data := map[string]interface{}{
			"js": []string{"main.js"},
		}
		s, err := r.Render(data, "index.html", "header.html")
		if err != nil {
			return err
		}
		return res.WriteHTML(s)
	}
}

func getMapHandler(_ *App, r *staticfiles.Renderer) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		data := map[string]interface{}{
			"js": []string{"map.js"},
		}
		s, err := r.Render(data, "map.html", "header.html")
		if err != nil {
			return err
		}
		return res.WriteHTML(s)
	}
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
		r := make(map[string]interface{}, 0)
		r["units"] = getUnits(app)
		r["messages"] = app.messages

		return res.WriteJSON(r)
	}
}

func getStackHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		return pprof.Lookup("goroutine").WriteTo(res.Body, 1)
	}
}

func getUnits(app *App) []*model.WebUnit {
	units := make([]*model.WebUnit, 0)

	app.units.Range(func(key, value interface{}) bool {
		switch v := value.(type) {
		case *model.Unit:
			units = append(units, v.ToWeb())
		case *model.Contact:
			units = append(units, v.ToWeb())
		case *model.Point:
			units = append(units, v.ToWeb())
		}
		return true
	})

	return units
}

func getConnHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		conn := make([]*Connection, 0)

		app.handlers.Range(func(key, value interface{}) bool {
			if v, ok := value.(*ClientHandler); ok {
				c := &Connection{
					Uid:  v.uid,
					User: v.user,
					Ssl:  v.ssl,
					Ver:  v.ver,
				}
				conn = append(conn, c)
			}
			return true
		})

		return res.WriteJSON(conn)
	}
}
