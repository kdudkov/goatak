package main

import (
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/kdudkov/goatak/pkg/model"
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

type Listener interface {
	Listen() error
	Address() string
}

type HttpServer struct {
	log       *slog.Logger
	listeners map[string]Listener
}

func NewHttp(app *App) *HttpServer {
	srv := &HttpServer{
		log:       app.logger.With("logger", "http"),
		listeners: make(map[string]Listener),
	}

	if addr := app.config.k.String("admin_addr"); addr != "" {
		srv.listeners["admin api calls"] = NewAdminAPI(app, addr, app.config.k.String("webtak_root"))
	}

	if addr := app.config.k.String("cert_addr"); addr != "" {
		srv.listeners["cert api calls"] = NewCertAPI(app, addr)
	}

	srv.listeners["marti api calls"] = NewMartiApi(app, app.config.k.String("api_addr"))

	return srv
}

func (h *HttpServer) Start() {
	for name, listener := range h.listeners {
		go func(name string, listener Listener) {
			h.log.Info(fmt.Sprintf("listening %s at %s", name, listener.Address()))

			if err := listener.Listen(); err != nil {
				h.log.Error("error", slog.Any("error", err))
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
