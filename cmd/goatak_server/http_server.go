package main

import (
	"embed"
	"fmt"
	"time"

	"github.com/aofei/air"

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
	Scope    string            `json:"scope"`
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
	_ = renderer.Load(templates)

	srv := &HttpServer{
		app:       app,
		listeners: make(map[string]*air.Air),
		renderer:  renderer,
	}

	if app.config.adminAddr != "" {
		srv.listeners["admin api calls"] = getAdminAPI(app, app.config.adminAddr, renderer, app.config.webtakRoot)
	}

	if app.config.certAddr != "" {
		srv.listeners["cert api calls"] = getCertAPI(app, app.config.certAddr)
	}

	srv.listeners["marti api calls"] = getMartiApi(app, app.config.apiAddr)

	return srv
}

func (h *HttpServer) Start() {
	for name, listener := range h.listeners {
		go func(name string, listener *air.Air) {
			h.app.Logger.Info(fmt.Sprintf("listening %s at %s", name, listener.Address))

			if err := listener.Serve(); err != nil {
				h.app.Logger.Error("error", "error", err.Error())
				panic(err)
			}
		}(name, listener)
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
