package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/kdudkov/goatak/internal/repository"
	"github.com/kdudkov/goatak/pkg/model"
)

//go:embed templates
var templates embed.FS

type Connection struct {
	Addr     string            `json:"addr"`
	User     string            `json:"user"`
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
	log         *slog.Logger
	listeners   map[string]Listener
	userManager repository.AuthRepository
	tokenKey    []byte
	tokenMaxAge time.Duration
	loginUrl    string
	noAuth      []string
}

func NewHttp(app *App) *HttpServer {
	mac := hmac.New(sha512.New, []byte(randomString(25)))

	srv := &HttpServer{
		log:         app.logger.With("logger", "http"),
		listeners:   make(map[string]Listener),
		userManager: app.users,
		tokenKey:    mac.Sum(nil),
		tokenMaxAge: time.Hour * 48,
		loginUrl:    "/login",
		noAuth:      []string{"/cot_xml"},
	}

	if addr := app.config.String("admin_addr"); addr != "" {
		srv.NewAdminAPI(app, addr, app.config.String("webtak_root"))
	}

	if addr := app.config.String("cert_addr"); addr != "" {
		srv.NewCertAPI(app, addr)
	}

	srv.NewMartiAPI(app, app.config.String("api_addr"))
	srv.NewLocalAPI(app, app.config.String("local_addr"))

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
