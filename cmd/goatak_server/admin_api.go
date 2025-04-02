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
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/template/html/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kdudkov/goatak/cmd/goatak_server/tak_ws"
	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/internal/wshandler"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/log"
	"github.com/kdudkov/goatak/pkg/model"
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

	api.f = fiber.New(fiber.Config{EnablePrintRoutes: false, DisableStartupMessage: true, Views: engine})

	api.f.Use(log.NewFiberLogger(&log.LoggerConfig{Name: "admin_api", Level: slog.LevelDebug}))

	staticfiles.Embed(api.f)

	api.f.Get("/", getIndexHandler())
	api.f.Get("/units", getUnitsHandler())
	api.f.Get("/map", getMapHandler())
	api.f.Get("/missions", getMissionsPageHandler())
	api.f.Get("/files", getFilesPage()).Name("admin_files")
	api.f.Get("/points", getPointsPage())
	api.f.Get("/devices", getDevicesPage())

	api.f.Get("/api/config", getConfigHandler(app))
	api.f.Get("/api/connections", getApiConnHandler(app))

	api.f.Get("/api/unit", getApiUnitsHandler(app))
	api.f.Get("/api/unit/:uid/track", getApiUnitTrackHandler(app))
	api.f.Delete("/api/unit/:uid", deleteItemHandler(app))
	api.f.Get("/api/message", getMessagesHandler(app))

	api.f.Get("/ws", getWsHandler(app))
	api.f.Get("/takproto/1", getTakWsHandler(app))
	api.f.Post("/cot", getCotPostHandler(app))
	api.f.Post("/cot_xml", getCotXMLPostHandler(app))

	api.f.Get("/api/file", getApiFilesHandler(app))
	api.f.Get("/api/file/:id", GetApiFileHandler(app))
	api.f.Get("/api/file/delete/:id", getApiFileDeleteHandler(app))
	api.f.Get("/api/point", getApiPointsHandler(app))
	api.f.Get("/api/device", getApiDevicesHandler(app))
	api.f.Put("/api/device/:id", getApiDevicePutHandler(app))

	api.f.Get("/api/mission", getApiAllMissionHandler(app))
	api.f.Get("/api/mission/:id/changes", getApiAllMissionChangesHandler(app))

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

func getUnitsHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " units",
			"js":    []string{"util.js", "units.js"},
		}

		return ctx.Render("templates/units", data, "templates/menu", "templates/header")
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

func getFilesPage() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " files",
			"js":    []string{"files.js"},
		}

		return ctx.Render("templates/files", data, "templates/menu", "templates/header")
	}
}

func getPointsPage() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " points",
			"js":    []string{"points.js"},
		}

		return ctx.Render("templates/points", data, "templates/menu", "templates/header")
	}
}

func getDevicesPage() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " devices",
			"js":    []string{"devices.js"},
		}

		return ctx.Render("templates/devices", data, "templates/menu", "templates/header")
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

func getApiUnitsHandler(app *App) fiber.Handler {
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
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})

	return adaptor.HTTPHandler(handler)
}

func getApiUnitTrackHandler(app *App) fiber.Handler {
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

func getApiConnHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		conn := make([]*Connection, 0)

		app.ForAllClients(func(ch client.ClientHandler) bool {
			c := &Connection{
				Uids:     ch.GetUids(),
				User:     ch.GetDevice().GetLogin(),
				Ver:      ch.GetVersion(),
				Addr:     ch.GetName(),
				Scope:    ch.GetDevice().GetScope(),
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

func getApiAllMissionHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.dbm.MissionQuery().Full().Get()

		result := make([]*model.MissionDTO, len(data))

		for i, m := range data {
			result[i] = model.ToMissionDTOAdm(m)
		}

		return ctx.JSON(result)
	}
}

func getApiAllMissionChangesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		id, err := ctx.ParamsInt("id")
		if err != nil {
			return err
		}

		m := app.dbm.MissionQuery().Id(uint(id)).One()

		ch := app.dbm.GetChanges(m.ID, time.Now().Add(-time.Hour*24*365), false)

		result := make([]*model.MissionChangeDTO, len(ch))

		for i, c := range ch {
			result[i] = model.ToChangeDTO(c, m.Name)
		}

		return ctx.JSON(result)
	}
}

func getApiFilesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.dbm.ResourceQuery().Order("created_at DESC").Get()

		return ctx.JSON(data)
	}
}

func GetApiFileHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		id, err := ctx.ParamsInt("id")

		if err != nil {
			return err
		}

		if id == 0 {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		pi := app.dbm.ResourceQuery().Id(uint(id)).One()

		if pi == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		f, err := app.files.GetFile(pi.Hash, pi.Scope)

		if err != nil {
			app.logger.Error("get file error", slog.Any("error", err))
			return err
		}

		defer f.Close()

		ctx.Set(fiber.HeaderETag, pi.Hash)
		ctx.Set(fiber.HeaderContentType, pi.MIMEType)
		ctx.Set(fiber.HeaderLastModified, pi.CreatedAt.UTC().Format(http.TimeFormat))
		ctx.Set(fiber.HeaderContentLength, strconv.Itoa(pi.Size))

		if !strings.HasPrefix(pi.MIMEType, "image/") {
			fn := pi.FileName
			if pi.MIMEType == "application/x-zip-compressed" && !strings.HasSuffix(fn, ".zip") {
				fn += ".zip"
			}
			ctx.Set(fiber.HeaderContentDisposition, "attachment; filename="+fn)
		}

		_, err = io.Copy(ctx, f)

		return err
	}
}

func getApiFileDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		id, err := ctx.ParamsInt("id")

		if err != nil {
			return err
		}

		if id == 0 {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		app.dbm.ResourceQuery().Id(uint(id)).Delete()

		return ctx.RedirectToRoute("admin_files", nil)
	}
}

func getApiPointsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.dbm.PointQuery().Order("created_at DESC").Get()

		return ctx.JSON(data)
	}
}

func getApiDevicesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.dbm.DeviceQuery().Get()

		devices := make([]*model.DeviceDTO, len(data))

		for i, d := range data {
			devices[i] = d.DTO()
		}

		return ctx.JSON(devices)
	}
}

func getApiDevicePutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		login := ctx.Params("id")

		d := app.dbm.DeviceQuery().Login(login).One()

		if d == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		var m *model.DevicePutDTO

		if err := ctx.BodyParser(&m); err != nil {
			return err
		}

		if m.Password != "" {
			if err := d.SetPassword(m.Password); err != nil {
				return err
			}
		}

		if m.Scope != "" {
			d.Scope = m.Scope
		}

		d.Callsign = m.Callsign
		d.Team = m.Team
		d.Role = m.Role

		app.dbm.Save(d)

		return ctx.JSON(fiber.Map{"status": "ok"})
	}
}

func getWsHandler(app *App) fiber.Handler {
	return websocket.New(func(ws *websocket.Conn) {
		name := uuid.NewString()

		h := wshandler.NewHandler(app.logger, name, ws)

		app.logger.Debug("ws listener connected")
		app.items.ChangeCallback().SubscribeNamed(name, h.SendItem)
		app.items.DeleteCallback().SubscribeNamed(name, h.DeleteItem)
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
			"url":     "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
			"maxzoom": 19,
			"parts":   []string{"a", "b", "c"},
		},
		{
			"name":    "Opentopo.cz",
			"url":     "https://tile-{s}.opentopomap.cz/{z}/{x}/{y}.png",
			"maxzoom": 18,
			"parts":   []string{"a", "b", "c"},
		},
		{
			"name":    "Google Hybrid",
			"url":     "http://mt{s}.google.com/vt/lyrs=y&x={x}&y={y}&z={z}&s=Galileo&scale=2",
			"maxzoom": 20,
			"parts":   []string{"0", "1", "2", "3"},
		},
		{
			"name":    "Yandex maps",
			"url":     "https://core-renderer-tiles.maps.yandex.net/tiles?l=map&x={x}&y={y}&z={z}&scale=2&lang=ru_RU&projection=web_mercator",
			"maxzoom": 20,
		},
	}
}
