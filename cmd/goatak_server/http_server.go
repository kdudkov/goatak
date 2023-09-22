package main

import (
	"embed"
	"errors"
	"net/http"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/aofei/air"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
	"github.com/kdudkov/goatak/staticfiles"
)

//go:embed templates
var templates embed.FS

type Connection struct {
	Addr     string            `json:"addr"`
	User     string            `json:"user"`
	Ssl      bool              `json:"ssl"`
	Ver      int32             `json:"ver"`
	Uids     map[string]string `json:"uids"`
	LastSeen *time.Time        `json:"last_seen"`
}

type HttpServer struct {
	app       *App
	listeners map[string]*air.Air
	renderer  *staticfiles.Renderer
}

func NewHttp(app *App) *HttpServer {
	renderer := new(staticfiles.Renderer)
	renderer.LeftDelimeter = "[["
	renderer.RightDelimeter = "]]"
	renderer.Load(templates, "templates")

	srv := &HttpServer{
		app:       app,
		listeners: make(map[string]*air.Air),
		renderer:  renderer,
	}

	if app.config.adminAddr != "" {
		srv.listeners["admin api calls"] = getAdminApi(app, app.config.adminAddr, renderer, app.config.webtakRoot)
	}
	if app.config.certAddr != "" {
		srv.listeners["cert api calls"] = getCertApi(app, app.config.certAddr)
	}
	srv.listeners["marti api calls"] = getMartiApi(app, app.config.apiAddr)

	return srv
}

func getAdminApi(app *App, addr string, renderer *staticfiles.Renderer, webtakRoot string) *air.Air {
	adminApi := air.New()
	adminApi.Address = addr

	staticfiles.EmbedFiles(adminApi, "/static")
	adminApi.GET("/", getIndexHandler(app, renderer))
	adminApi.GET("/map", getMapHandler(app, renderer))
	adminApi.GET("/config", getConfigHandler(app))
	adminApi.GET("/connections", getConnHandler(app))

	adminApi.GET("/unit", getUnitsHandler(app))
	adminApi.DELETE("/unit/:uid", deleteItemHandler(app))

	adminApi.GET("/takproto/1", getWsHandler(app))

	if webtakRoot != "" {
		adminApi.FILE("/webtak/", filepath.Join(webtakRoot, "index.html"))
		adminApi.FILES("/webtak", webtakRoot)
		addMartiRoutes(app, adminApi)
	}

	adminApi.GET("/stack", getStackHandler())

	adminApi.RendererTemplateLeftDelim = "[["
	adminApi.RendererTemplateRightDelim = "]]"

	return adminApi
}

func (h *HttpServer) Start() {
	for name, listener := range h.listeners {
		go func(name string, listener *air.Air) {
			h.app.Logger.Infof("listening %s at %s", name, listener.Address)
			if err := listener.Serve(); err != nil {
				h.app.Logger.Panicf(err.Error())
			}
		}(name, listener)
	}
}

func getIndexHandler(app *App, r *staticfiles.Renderer) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		data := map[string]any{
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
		data := map[string]any{
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
	m := make(map[string]any, 0)
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
		r := make(map[string]any, 0)
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

	app.items.ForEach(func(item *model.Item) bool {
		units = append(units, item.ToWeb())
		return true
	})

	return units
}

func deleteItemHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		uid := getStringParam(req, "uid")
		app.items.Remove(uid)

		r := make(map[string]any, 0)
		r["units"] = getUnits(app)
		r["messages"] = app.messages
		return res.WriteJSON(r)
	}
}

func getConnHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		conn := make([]*Connection, 0)

		app.ForAllClients(func(ch cot.ClientHandler) bool {
			c := &Connection{
				Uids:     ch.GetUids(),
				User:     ch.GetUser(),
				Ver:      ch.GetVersion(),
				Addr:     ch.GetName(),
				LastSeen: ch.GetLastSeen(),
			}
			conn = append(conn, c)
			return true
		})

		return res.WriteJSON(conn)
	}
}
