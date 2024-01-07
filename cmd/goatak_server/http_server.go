package main

import (
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"time"

	"github.com/aofei/air"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kdudkov/goatak/internal/client"
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

func getAdminAPI(app *App, addr string, renderer *staticfiles.Renderer, webtakRoot string) *air.Air {
	adminAPI := air.New()
	adminAPI.Address = addr
	adminAPI.NotFoundHandler = getNotFoundHandler()
	// adminAPI.Gases = []air.Gas{LoggerGas(app.Logger, "admin_api")}

	staticfiles.EmbedFiles(adminAPI, "/static")
	adminAPI.GET("/", getIndexHandler(app, renderer))
	adminAPI.GET("/map", getMapHandler(app, renderer))
	adminAPI.GET("/config", getConfigHandler(app))
	adminAPI.GET("/connections", getConnHandler(app))

	adminAPI.GET("/unit", getUnitsHandler(app))
	adminAPI.GET("/unit/:uid/track", getUnitTrackHandler(app))
	adminAPI.DELETE("/unit/:uid", deleteItemHandler(app))

	adminAPI.GET("/takproto/1", getWsHandler(app))
	adminAPI.POST("/cot", getCotPostHandler(app))
	adminAPI.POST("/cot_xml", getCotXMLPostHandler(app))

	if webtakRoot != "" {
		adminAPI.FILE("/webtak/", filepath.Join(webtakRoot, "index.html"))
		adminAPI.FILES("/webtak", webtakRoot)
		addMartiRoutes(app, adminAPI)
	}

	adminAPI.GET("/stack", getStackHandler())
	adminAPI.GET("/metrics", getMetricsHandler())

	adminAPI.RendererTemplateLeftDelim = "[["
	adminAPI.RendererTemplateRightDelim = "]]"

	return adminAPI
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
			app.Logger.Errorf("%v", err)
			_ = res.WriteString(err.Error())

			return err
		}

		return res.WriteHTML(s)
	}
}

func getMapHandler(app *App, r *staticfiles.Renderer) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		data := map[string]any{
			"js": []string{"map.js"},
		}

		s, err := r.Render(data, "map.html", "header.html")
		if err != nil {
			app.Logger.Errorf("%v", err)
			_ = res.WriteString(err.Error())

			return err
		}

		return res.WriteHTML(s)
	}
}

func getNotFoundHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		res.Status = http.StatusNotFound

		return errors.New(http.StatusText(res.Status))
	}
}

func getConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	m := make(map[string]any, 0)
	m["lat"] = app.lat
	m["lon"] = app.lon
	m["zoom"] = app.zoom
	m["version"] = getVersion()

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

func getMetricsHandler() func(req *air.Request, res *air.Response) error {
	h := promhttp.Handler()

	return func(req *air.Request, res *air.Response) error {
		h.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())

		return nil
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

func getUnitTrackHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		uid := getStringParam(req, "uid")

		item := app.items.Get(uid)
		if item == nil {
			res.Status = http.StatusNotFound

			return nil
		}

		return res.WriteJSON(item.GetTrack())
	}
}

func deleteItemHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		uid := getStringParam(req, "uid")
		app.items.Remove(uid)
		app.missions.DeletePoint(uid)

		r := make(map[string]any, 0)
		r["units"] = getUnits(app)
		r["messages"] = app.messages

		return res.WriteJSON(r)
	}
}

func getConnHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		conn := make([]*Connection, 0)

		app.ForAllClients(func(ch client.ClientHandler) bool {
			c := &Connection{
				Uids:     ch.GetUids(),
				User:     ch.GetUser().GetLogin(),
				Ver:      ch.GetVersion(),
				Addr:     ch.GetName(),
				Scope:    ch.GetUser().GetScope(),
				LastSeen: ch.GetLastSeen(),
			}
			conn = append(conn, c)

			return true
		})

		sort.Slice(conn, func(i, j int) bool {
			return conn[i].Addr < conn[j].Addr
		})

		return res.WriteJSON(conn)
	}
}

func getCotPostHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		c := new(cot.CotMessage)

		dec := json.NewDecoder(req.Body)

		if err := dec.Decode(c); err != nil {
			app.Logger.Errorf("cot decode error %s", err)

			return err
		}

		app.NewCotMessage(c)

		return nil
	}
}

func getCotXMLPostHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		scope := getStringParam(req, "scope")
		if scope == "" {
			scope = "test"
		}

		ev := new(cot.Event)

		dec := xml.NewDecoder(req.Body)

		if err := dec.Decode(ev); err != nil {
			app.Logger.Errorf("cot decode error %s", err)

			return err
		}

		c, err := cot.EventToProto(ev)
		if err != nil {
			app.Logger.Errorf("cot convert error %s", err)

			return err
		}

		c.Scope = scope
		app.NewCotMessage(c)

		return nil
	}
}
