package main

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"log/slog"
	"net/http"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/template/html/v2"
	"github.com/google/uuid"
	"github.com/kdudkov/goatak/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kdudkov/goatak/cmd/goatak_server/tak_ws"
	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/internal/wshandler"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/staticfiles"
)

type AdminAPI struct {
	f    *fiber.App
	addr string
}

func NewAdminAPI(app *App, addr string, webtakRoot string) *AdminAPI {
	api := &AdminAPI{addr: addr}

	engine := html.NewFileSystem(http.FS(templates), ".html")

	engine.Delims("[[", "]]")

	api.f = fiber.New(fiber.Config{EnablePrintRoutes: false, DisableStartupMessage: true, Views: engine, BodyLimit: 64 * 1024 * 1024})

	api.f.Use(log.NewFiberLogger(&log.LoggerConfig{Name: "admin_api", Level: slog.LevelDebug}))

	staticfiles.Embed(api.f)

	api.f.Get("/", getIndexHandler())
	api.f.Get("/points", getPointsHandler())
	api.f.Get("/map", getMapHandler())
	api.f.Get("/missions", getMissionsPageHandler())
	api.f.Get("/packages", getMPPageHandler())
	api.f.Get("/config", getConfigHandler(app))
	api.f.Get("/connections", getConnHandler(app))

	api.f.Get("/unit", getUnitsHandler(app))
	api.f.Get("/unit/:uid/track", getUnitTrackHandler(app))
	api.f.Delete("/unit/:uid", deleteItemHandler(app))
	api.f.Get("/message", getMessagesHandler(app))

	api.f.Get("/ws", getWsHandler(app))
	api.f.Get("/takproto/1", getTakWsHandler(app))
	api.f.Post("/cot", getCotPostHandler(app))
	api.f.Post("/cot_xml", getCotXMLPostHandler(app))

	api.f.Get("/mp", getAllMissionPackagesHandler(app))
	api.f.Get("/mp/:uid", getPackageHandler(app))

	if app.missions != nil {
		api.f.Get("/mission", getAllMissionHandler(app))
	}

	if webtakRoot != "" {
		api.f.Static("/webtak", webtakRoot)

		addMartiRoutes(app, api.f)
	}

	api.f.Get("/stack", getStackHandler())
	api.f.Get("/metrics", getMetricsHandler())

	return api
}

func (api *AdminAPI) Address() string {
	return api.addr
}

func (api *AdminAPI) Listen() error {
	return api.f.Listen(api.addr)
}

func getIndexHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " dash",
			"js":    []string{"util.js", "main.js"},
		}

		return ctx.Render("templates/index", data, "templates/menu", "templates/header")
	}
}

func getPointsHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " points",
			"js":    []string{"util.js", "points.js"},
		}

		return ctx.Render("templates/points", data, "templates/menu", "templates/header")
	}
}

func getMapHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"js":    []string{"util.js", "map.js"},
		}

		return ctx.Render("templates/map", data, "templates/header")
	}
}

func getMissionsPageHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " missions",
			"js":    []string{"missions.js"},
		}

		return ctx.Render("templates/missions", data, "templates/menu", "templates/header")
	}
}

func getMPPageHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " mp",
			"js":    []string{"mp.js"},
		}

		return ctx.Render("templates/mp", data, "templates/menu", "templates/header")
	}
}

func getConfigHandler(app *App) fiber.Handler {
	m := make(map[string]any, 0)
	m["lat"] = app.lat
	m["lon"] = app.lon
	m["zoom"] = app.zoom
	m["version"] = getVersion()

	m["layers"] = getDefaultLayers()

	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(m)
	}
}

func getUnitsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(getUnits(app))
	}
}

func getMessagesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(app.messages)
	}
}

func getStackHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return pprof.Lookup("goroutine").WriteTo(ctx.Response().BodyWriter(), 1)
	}
}

func getMetricsHandler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{DisableCompression: true},
	))
}

func getUnitTrackHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Params("uid")

		item := app.items.Get(uid)
		if item == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(item.GetTrack())
	}
}

func deleteItemHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Params("uid")
		app.items.Remove(uid)

		r := make(map[string]any, 0)
		r["units"] = getUnits(app)
		r["messages"] = app.messages

		return ctx.JSON(r)
	}
}

func getConnHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
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

		return ctx.JSON(conn)
	}
}

func getCotPostHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		c := new(cot.CotMessage)

		if err := json.Unmarshal(ctx.Body(), c); err != nil {
			app.logger.Error("cot decode error", slog.Any("error", err))

			return err
		}

		app.NewCotMessage(c)

		return nil
	}
}

func getCotXMLPostHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		scope := ctx.Query("scope")
		if scope == "" {
			scope = "test"
		}

		ev := new(cot.Event)

		if err := xml.Unmarshal(ctx.Body(), &ev); err != nil {
			app.logger.Error("cot decode error", slog.Any("error", err))

			return err
		}

		c, err := cot.EventToProto(ev)
		if err != nil {
			app.logger.Error("cot convert error", slog.Any("error", err))

			return err
		}

		c.Scope = scope
		app.NewCotMessage(c)

		return nil
	}
}

func getAllMissionHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.missions.GetAllMissionsAdm()

		result := make([]*model.MissionDTO, len(data))

		for i, m := range data {
			result[i] = model.ToMissionDTOAdm(m, app.packageManager)
		}

		return ctx.JSON(result)
	}
}

func getAllMissionPackagesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.packageManager.GetList(nil)

		return ctx.JSON(data)
	}
}

func getPackageHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		pi := app.packageManager.Get(ctx.Params("uid"))

		if pi == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		f, err := app.packageManager.GetFile(pi.Hash)

		if err != nil {
			app.logger.Error("get file error", slog.Any("error", err))
			return err
		}

		defer f.Close()

		ctx.Set(fiber.HeaderContentType, pi.MIMEType)

		if !strings.HasPrefix(pi.MIMEType, "image/") {
			fn := pi.Name
			if pi.MIMEType == "application/x-zip-compressed" && !strings.HasSuffix(fn, ".zip") {
				fn += ".zip"
			}
			ctx.Set(fiber.HeaderContentDisposition, "attachment; filename="+fn)
		}

		ctx.Set("Last-Modified", pi.SubmissionDateTime.UTC().Format(http.TimeFormat))
		ctx.Set("Content-Length", strconv.Itoa(pi.Size))

		_, err = io.Copy(ctx, f)

		return err
	}
}

func getWsHandler(app *App) fiber.Handler {
	return websocket.New(func(ws *websocket.Conn) {
		name := uuid.NewString()

		h := wshandler.NewHandler(app.logger, name, ws)

		app.logger.Debug("ws listener connected")
		app.changeCb.SubscribeNamed(name, h.SendItem)
		app.deleteCb.SubscribeNamed(name, h.DeleteItem)
		h.Listen()
		app.logger.Debug("ws listener disconnected")
	})
}

// handler for WebTAK client - sends/receives protobuf COTs
func getTakWsHandler(app *App) fiber.Handler {
	return websocket.New(func(ws *websocket.Conn) {
		defer ws.Close()

		app.logger.Info("WS connection from " + ws.RemoteAddr().String())
		name := "ws:" + ws.RemoteAddr().String()
		w := tak_ws.New(name, nil, ws, app.NewCotMessage)

		app.AddClientHandler(w)
		w.Listen()
		app.logger.Info("ws disconnected")
		app.RemoveClientHandler(w.GetName())
	})
}

func getDefaultLayers() []map[string]any {
	return []map[string]any{
		{
			"name":    "OSM",
			"url":     "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
			"maxzoom": 19,
		},
		{
			"name":    "Opentopo.cz",
			"url":     "https://tile-{s}.opentopomap.cz/{z}/{x}/{y}.png",
			"maxzoom": 18,
			"parts":   []string{"a", "b", "c"},
		},
		{
			"name":    "Google Hybrid",
			"url":     "http://mt{s}.google.com/vt/lyrs=y&x={x}&y={y}&z={z}&s=Galileo",
			"maxzoom": 20,
			"parts":   []string{"0", "1", "2", "3"},
		},
		{
			"name":    "Yandex maps",
			"url":     "https://core-renderer-tiles.maps.yandex.net/tiles?l=map&x={x}&y={y}&z={z}&scale=1&lang=ru_RU&projection=web_mercator",
			"maxzoom": 20,
		},
	}
}
