package main

import (
	"encoding/xml"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/google/uuid"

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

func (h *HttpServer) NewAdminAPI(app *App, addr string, webtakRoot string) *AdminAPI {
	api := new(AdminAPI)
	api.addr = addr
	h.listeners["admin api calls"] = api

	engine := html.NewFileSystem(http.FS(templates), ".html")

	engine.Delims("[[", "]]")

	api.f = fiber.New(fiber.Config{EnablePrintRoutes: false, DisableStartupMessage: true, Views: engine})

	api.f.Use(log.NewFiberLogger(&log.LoggerConfig{Name: "admin_api", Level: slog.LevelDebug, UserGetter: Username}))
	api.f.Use(h.CookieAuth)

	staticfiles.Embed(api.f)

	api.f.Get("/login", h.getAdminLoginHandler(app.config.Bool("delay")))
	api.f.Post("/login", h.getAdminLoginHandler(app.config.Bool("delay")))
	api.f.Post("/token", h.getAdminTokenHandler())
	api.f.Get("/logout", logoutHandler)

	api.f.Get("/", getIndexHandler())
	api.f.Get("/units", getUnitsHandler())
	api.f.Get("/map", getMapHandler())
	api.f.Get("/missions", getMissionsPageHandler())
	api.f.Get("/files", getFilesPage()).Name("admin_files")
	api.f.Get("/points", getPointsPage())
	api.f.Get("/devices", getDevicesPage())
	api.f.Get("/profiles", getProfilesPage())
	api.f.Get("/feeds", getFeedsPage())

	api.f.Get("/api/config", getConfigHandler(app))
	api.f.Get("/api/connections", getApiConnHandler(app))

	api.f.Get("/api/unit", getApiUnitsHandler(app))
	api.f.Get("/api/unit/:uid/track", getApiUnitTrackHandler(app))
	api.f.Delete("/api/unit/:uid", deleteItemHandler(app))

	api.f.Get("/ws", getWsHandler(app))
	api.f.Get("/takproto/1", getTakWsHandler(app))
	api.f.Post("/cot", getCotPostHandler(app))
	api.f.Post("/cot_xml", getCotXMLPostHandler(app))

	api.f.Get("/api/file", getApiFilesHandler(app))
	api.f.Get("/api/file/:id/zip", getApiZipFileHandler(app))
	api.f.Get("/api/file/:id", getApiFileHandler(app))
	api.f.Get("/api/file/delete/:id", getApiFileDeleteHandler(app))

	api.f.Get("/api/point", getApiPointsHandler(app))
	api.f.Get("/api/device", getApiDevicesHandler(app))
	api.f.Post("/api/device", getApiDevicePostHandler(app))
	api.f.Put("/api/device/:id", getApiDevicePutHandler(app))
	api.f.Get("/api/cert", getApiCertsHandler(app))
	api.f.Get("/api/profile", getApiProfilesHandler(app))
	api.f.Post("/api/profile", getApiProfilePostHandler(app))
	api.f.Put("/api/profile/:login/:uid", getApiProfilePutHandler(app))
	api.f.Delete("/api/profile/:login/:uid", getApiProfileDeleteHandler(app))

	api.f.Get("/api/feed", getApiFeedsHandler(app))
	api.f.Post("/api/feed", getApiFeedPostHandler(app))
	api.f.Put("/api/feed/:uid", getApiFeedPutHandler(app))
	api.f.Delete("/api/feed/:uid", getApiFeedDeleteHandler(app))

	api.f.Get("/api/mission", getApiAllMissionHandler(app))
	api.f.Get("/api/mission/:id/changes", getApiAllMissionChangesHandler(app))

	if webtakRoot != "" {
		api.f.Static("/webtak", webtakRoot)
		api.f.Get("/webtak-plugins/webtak-manifest.json", getPluginsManifestHandler(app))
		addMartiRoutes(app, api.f)
	}

	return api
}

func (api *AdminAPI) Address() string {
	return api.addr
}

func (api *AdminAPI) Listen() error {
	return api.f.Listen(api.addr)
}

func (h *HttpServer) getAdminLoginHandler(delay bool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		login := c.FormValue("login")

		if login == "" {
			return c.Render("templates/login", nil)
		}

		if user := h.userManager.Get(login); user.CanLogIn() && user.CheckPassword(c.FormValue("password")) {
			token, err := generateToken(login, h.tokenKey, h.tokenMaxAge)

			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
			}

			cookie := &fiber.Cookie{Name: cookieName,
				Value: token, Secure: false, HTTPOnly: true, Expires: time.Now().Add(h.tokenMaxAge)}
			c.Cookie(cookie)

			return c.Redirect("/")
		}

		h.log.Warn("invalid login", "user", login)

		if delay {
			time.Sleep(time.Second * time.Duration(1+rand.Intn(5)))
		}

		return c.Render("templates/login", fiber.Map{"login": login, "error": "bad login or password"})
	}
}

func (h *HttpServer) getAdminTokenHandler() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		m := make(map[string]string)

		if err := c.BodyParser(&m); err != nil {
			return err
		}

		if login := m["login"]; login != "" {
			if user := h.userManager.Get(login); user != nil {
				if user.CanLogIn() && user.CheckPassword(m["password"]) {
					token, err := generateToken(login, h.tokenKey, h.tokenMaxAge)

					if err != nil {
						h.log.Error("generate token error", slog.Any("error", err))
						return err
					}

					return c.JSON(fiber.Map{"token": token})
				}
			}

			h.log.Warn("invalid login", "user", login)
		}

		return c.SendStatus(fiber.StatusUnauthorized)
	}
}

func logoutHandler(c *fiber.Ctx) error {
	c.ClearCookie(cookieName)

	return c.Redirect("/")
}

func getIndexHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " dash",
			"js":    []string{"main.js"},
		}

		return ctx.Render("templates/index", data, "templates/menu", "templates/header")
	}
}

func getUnitsHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " units",
			"js":    []string{"units.js"},
		}

		return ctx.Render("templates/units", data, "templates/menu", "templates/header")
	}
}

func getMapHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"js":    []string{"map.js"},
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

func getProfilesPage() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " profiles",
			"js":    []string{"profiles.js"},
		}

		return ctx.Render("templates/profiles", data, "templates/menu", "templates/header")
	}
}

func getFeedsPage() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := map[string]any{
			"theme": "auto",
			"page":  " feeds",
			"js":    []string{"feeds.js"},
		}

		return ctx.Render("templates/feeds", data, "templates/menu", "templates/header")
	}
}

func getConfigHandler(app *App) fiber.Handler {
	m := make(map[string]any, 0)
	m["lat"] = app.lat
	m["lon"] = app.lon
	m["zoom"] = app.zoom
	m["version"] = getVersion()
	l, err := app.config.Layers()

	if err != nil {
		app.logger.Error("error loading layers", slog.Any("error", err))
	}

	m["layers"] = l

	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(m)
	}
}

func getApiUnitsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(getUnits(app))
	}
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

func getCotXMLPostHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		scope := ctx.Query("scope", "test")

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

		return ctx.JSON(model.MissionDTOList(m.Name, ch))
	}
}

func getApiFilesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.dbm.ResourceQuery().Order("created_at DESC").Get()

		return ctx.JSON(data)
	}
}

func getApiFileHandler(app *App) fiber.Handler {
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

func getApiZipFileHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		fname := ctx.Query("name")
		id, err := ctx.ParamsInt("id")

		if err != nil {
			return err
		}

		if id == 0 || fname == "" {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		pi := app.dbm.ResourceQuery().Id(uint(id)).One()

		if pi == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		if pi.MIMEType != "application/x-zip-compressed" {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		z, err := app.files.GetZipFile(pi.Hash, pi.Scope, fname)

		if err != nil {
			app.logger.Error("get file error", slog.Any("error", err))
			return err
		}

		ctx.Set(fiber.HeaderContentType, getMime(filepath.Ext(fname)))
		ctx.Set(fiber.HeaderLastModified, z.Modified.Format(http.TimeFormat))
		ctx.Set(fiber.HeaderContentLength, strconv.Itoa(int(z.UncompressedSize64)))
		ctx.Set(fiber.HeaderContentDisposition, "inline; filename="+filepath.Base(fname))

		f, err := z.Open()

		if err != nil {
			return err
		}

		defer f.Close()

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

		pi := app.dbm.ResourceQuery().Id(uint(id)).One()

		if pi == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}
		
		app.dbm.ResourceQuery().Id(uint(id)).Delete()
		app.files.Delete(pi.Hash, pi.Scope)
		
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
		data := app.dbm.DeviceQuery().Full().Get()

		devices := make([]*model.DeviceDTO, len(data))

		for i, d := range data {
			devices[i] = d.DTO()
		}

		return ctx.JSON(devices)
	}
}

func getApiDevicePostHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var m *model.DevicePostDTO

		if err := ctx.BodyParser(&m); err != nil {
			return err
		}

		if m.Login == "" {
			return SendError(ctx, "empty login")
		}

		if m.Password == "" {
			return SendError(ctx, "empty password")
		}

		if m.Scope == "" {
			return SendError(ctx, "empty scope")
		}

		d := &model.Device{
			Login:     m.Login,
			Admin:     m.Admin,
			Disabled:  m.Disabled,
			Scope:     m.Scope,
			ReadScope: m.ReadScope,
		}

		if err := d.SetPassword(m.Password); err != nil {
			return err
		}

		if err := app.dbm.Create(d); err != nil {
			return SendError(ctx, err.Error())
		}

		return ctx.JSON(d.DTO())
	}
}

func getApiCertsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.dbm.CertsQuery().Get()

		certs := make([]*model.CertificateDTO, len(data))

		for i, d := range data {
			certs[i] = d.DTO()
		}

		return ctx.JSON(certs)
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

		d.Scope = m.Scope
		d.ReadScope = m.ReadScope
		//d.Admin = m.Admin
		d.Disabled = m.Disabled

		app.dbm.Save(d)

		return ctx.JSON(d.DTO())
	}
}

func getApiProfilesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.dbm.ProfileQuery().Get()

		profiles := make([]*model.ProfileDTO, len(data))

		for i, p := range data {
			profiles[i] = p.DTO()
		}

		return ctx.JSON(profiles)
	}
}

func getApiProfilePostHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var m *model.ProfilePostDTO

		if err := ctx.BodyParser(&m); err != nil {
			return err
		}

		p := &model.Profile{
			Login:    m.Login,
			UID:      m.UID,
			Callsign: m.Callsign,
			Team:     m.Team,
			Role:     m.Role,
			CotType:  m.CotType,
			Options:  m.Options,
		}

		if p.Login == "" {
			p.Login = "*"
		}

		if p.UID == "" {
			p.UID = "*"
		}

		if err := app.dbm.Create(p); err != nil {
			return SendError(ctx, err.Error())
		}

		return ctx.JSON(p.DTO())
	}
}

func getApiProfilePutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		login := ctx.Params("login")
		uid := ctx.Params("uid")

		p := app.dbm.ProfileQuery().Login(login).UID(uid).One()

		if p == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		var m *model.ProfilePutDTO

		if err := ctx.BodyParser(&m); err != nil {
			return err
		}

		p.Callsign = m.Callsign
		p.Team = m.Team
		p.Role = m.Role
		p.CotType = m.CotType
		p.Options = m.Options

		if err := app.dbm.ForceSave(p); err != nil {
			return SendError(ctx, err.Error())
		}

		return ctx.JSON(p.DTO())
	}
}

func getApiProfileDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		login := ctx.Params("login")
		uid := ctx.Params("uid")

		if err := app.dbm.ProfileQuery().Login(login).UID(uid).Delete(); err != nil {
			return SendError(ctx, err.Error())
		}

		return ctx.JSON(fiber.Map{"status": "ok"})
	}
}

func getApiFeedsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := app.dbm.FeedQuery().All(true).Get()

		feeds := make([]*model.Feed2DTO, len(data))

		for i, f := range data {
			feeds[i] = f.DTO(true)
		}

		return ctx.JSON(feeds)
	}
}

func getApiFeedPostHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var m *model.FeedPostDTO

		if err := ctx.BodyParser(&m); err != nil {
			return err
		}

		if m.UID == "" {
			m.UID = uuid.NewString()
		}

		f := &model.Feed2{
			UID:       m.UID,
			Active:    m.Active,
			Alias:     m.Alias,
			URL:       m.URL,
			Latitude:  m.Latitude,
			Longitude: m.Longitude,
			Fov:       m.Fov,
			Heading:   m.Heading,
			Range:     m.Range,
			User:      Username(ctx),
			Scope:     m.Scope,
		}

		if err := app.dbm.Create(f); err != nil {
			return SendError(ctx, err.Error())
		}

		return ctx.JSON(f.DTO(true))
	}
}

func getApiFeedPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Params("uid")

		f := app.dbm.FeedQuery().UID(uid).All(true).One()

		if f == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		var m *model.FeedPutDTO

		if err := ctx.BodyParser(&m); err != nil {
			return err
		}

		f.Active = m.Active
		f.Alias = m.Alias
		f.URL = m.URL
		f.Latitude = m.Latitude
		f.Longitude = m.Longitude
		f.Fov = m.Fov
		f.Heading = m.Heading
		f.Range = m.Range
		f.Scope = m.Scope

		if err := app.dbm.Save(f); err != nil {
			return SendError(ctx, err.Error())
		}

		return ctx.JSON(f.DTO(true))
	}
}

func getApiFeedDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Params("uid")

		if err := app.dbm.FeedQuery().UID(uid).Delete(); err != nil {
			return SendError(ctx, err.Error())
		}

		return ctx.JSON(fiber.Map{"status": "ok"})
	}
}

func getPluginsManifestHandler(_ *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{"plugins": []string{}, "iconSets": []string{}})
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

// handler for WebTAK client - sends/receives protobuf COTs.
func getTakWsHandler(app *App) fiber.Handler {
	return websocket.New(func(ws *websocket.Conn) {
		defer ws.Close()

		app.logger.Info("WS connection from " + ws.RemoteAddr().String())
		name := "ws:" + ws.RemoteAddr().String()
		w := tak_ws.New(name, User(ws), ws, app.NewCotMessage)

		app.AddClientHandler(w)
		w.Listen()
		app.logger.Info("ws disconnected")
		app.RemoveHandlerCb(w)
	})
}

func getMime(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".kml":
		return "application/vnd.google-earth.kml+xml"
	case ".kmz":
		return "application/vnd.google-earth.kmz"
	case ".gpx":
		return "application/gpx+xml"
	case ".xml", ".cot":
		return "application/xml"
	case ".txt":
		return "text/txt"
	case ".json":
		return fiber.MIMEApplicationJSONCharsetUTF8
	default:
		return fiber.MIMEOctetStream
	}
}
