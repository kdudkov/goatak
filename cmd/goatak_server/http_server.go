package main

import (
	"crypto/tls"
	"embed"
	"errors"
	"net/http"
	"runtime/pprof"

	"github.com/aofei/air"
	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/model"
	"github.com/kdudkov/goatak/staticfiles"
)

//go:embed templates
var templates embed.FS

type Connection struct {
	Addr string            `json:"addr"`
	User string            `json:"user"`
	Ssl  bool              `json:"ssl"`
	Ver  int32             `json:"ver"`
	Uids map[string]string `json:"uids"`
}

type HttpServer struct {
	app       *App
	listeners map[string]*air.Air
	renderer  *staticfiles.Renderer
}

func NewHttp(app *App, adminAddress string, apiAddress string) *HttpServer {
	renderer := new(staticfiles.Renderer)
	renderer.LeftDelimeter = "[["
	renderer.RightDelimeter = "]]"
	renderer.Load(templates, "templates")

	srv := &HttpServer{
		app:       app,
		listeners: make(map[string]*air.Air),
		renderer:  renderer,
	}

	if adminAddress != "" {
		srv.listeners["admin api calls"] = getAdminApi(app, adminAddress, renderer)
	}
	srv.listeners["marti api calls"] = getMartiApi(app, apiAddress)
	//srv.listeners["tls api calls"] = getTlsApi(app, ":8446")

	return srv
}

func getAdminApi(app *App, addr string, renderer *staticfiles.Renderer) *air.Air {
	adminApi := air.New()
	adminApi.Address = addr

	staticfiles.EmbedFiles(adminApi, "/static")
	adminApi.GET("/", getIndexHandler(app, renderer))
	adminApi.GET("/map", getMapHandler(app, renderer))
	adminApi.GET("/config", getConfigHandler(app))
	adminApi.GET("/connections", getConnHandler(app))

	adminApi.GET("/unit", getUnitsHandler(app))
	adminApi.DELETE("/unit/:uid", deleteItemHandler(app))

	adminApi.GET("/stack", getStackHandler())

	adminApi.RendererTemplateLeftDelim = "[["
	adminApi.RendererTemplateRightDelim = "]]"

	return adminApi
}

func getTlsApi(app *App, addr string) *air.Air {
	tlsApi := air.New()
	tlsApi.Address = addr

	//auth := authenticator.BasicAuthGas(authenticator.BasicAuthGasConfig{
	//	Validator: func(username string, password string, _ *air.Request, _ *air.Response) (bool, error) {
	//		app.Logger.Infof("tls api login with user %s", username)
	//		return username == "user", nil
	//	},
	//})
	//
	//tlsApi.Gases = []air.Gas{auth}

	tlsApi.GET("/Marti/api/tls/config", getTlsConfigHandler(app))

	tlsApi.NotFoundHandler = getNotFoundHandler(app)

	if app.config.cert != nil {
		tlsCfg := &tls.Config{
			Certificates:       []tls.Certificate{*app.config.cert},
			ClientCAs:          app.config.ca,
			RootCAs:            app.config.ca,
			ClientAuth:         tls.NoClientCert,
			InsecureSkipVerify: true,
		}

		tlsApi.TLSConfig = tlsCfg
	}
	return tlsApi
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
		v := value.(*model.Item)
		units = append(units, v.ToWeb())
		return true
	})

	return units
}

func deleteItemHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		uid := getStringParam(req, "uid")
		app.units.Delete(uid)

		r := make(map[string]interface{}, 0)
		r["units"] = getUnits(app)
		r["messages"] = app.messages
		return res.WriteJSON(r)
	}
}

func getConnHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		conn := make([]*Connection, 0)

		app.handlers.Range(func(key, value interface{}) bool {
			if v, ok := value.(*cot.ClientHandler); ok {
				c := &Connection{
					Uids: v.GetUids(),
					User: v.GetUser(),
					Ver:  v.GetVersion(),
					Addr: v.GetName(),
				}
				conn = append(conn, c)
			}
			return true
		})

		return res.WriteJSON(conn)
	}
}
