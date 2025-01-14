package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kdudkov/goatak/cmd/goatak_server/mp"
	im "github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/internal/pm"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/log"
	"github.com/kdudkov/goatak/pkg/tools"

	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/kdudkov/goatak/pkg/model"
)

const (
	nodeID     = "1"
	apiVersion = "3"
)

type MartiAPI struct {
	f        *fiber.App
	addr     string
	tls      bool
	cert     tls.Certificate
	certPool *x509.CertPool
}

func NewMartiApi(app *App, addr string) *MartiAPI {
	api := &MartiAPI{
		f: fiber.New(fiber.Config{
			EnablePrintRoutes:     false,
			DisableStartupMessage: true,
			BodyLimit:             64 * 1024 * 1024,
			StreamRequestBody:     true}),
		addr: addr,
	}

	api.f.Use(NewMetricHandler("marti_api"))
	api.f.Use(log.NewFiberLogger(&log.LoggerConfig{Name: "marti_api", UserGetter: Username}))

	if app.config.UseSSL() {
		api.tls = true
		api.cert = *app.config.tlsCert
		api.certPool = app.config.certPool
		api.f.Use(SSLCheckHandler(app))
	}

	addMartiRoutes(app, api.f)

	return api
}

func (api *MartiAPI) Address() string {
	return api.addr
}

func (api *MartiAPI) Listen() error {
	if api.tls {
		return api.f.ListenMutualTLSWithCertificate(api.addr, api.cert, api.certPool)
	} else {
		return api.f.Listen(api.addr)
	}
}

func addMartiRoutes(app *App, f fiber.Router) {
	f.Get("/Marti/api/version", getVersionHandler(app))
	f.Get("/Marti/api/version/config", getVersionConfigHandler(app))
	f.Get("/Marti/api/clientEndPoints", getEndpointsHandler(app))
	f.Get("/Marti/api/contacts/all", getContactsHandler(app))

	f.Get("/Marti/api/util/user/roles", getUserRolesHandler(app))

	f.Get("/Marti/api/groups/all", getAllGroupsHandler(app))
	f.Get("/Marti/api/groups/groupCacheEnabled", getAllGroupsCacheHandler(app))

	f.Get("/Marti/api/device/profile/connection", getProfileConnectionHandler(app))

	f.Get("/Marti/sync/search", getSearchHandler(app))
	f.Get("/Marti/sync/missionquery", getMissionQueryHandler(app))
	f.Post("/Marti/sync/missionupload", getMissionUploadHandler(app))
	f.Get("/Marti/sync/content", getContentGetHandler(app))
	f.Post("/Marti/sync/upload", getUploadHandler(app))
	f.Get("/Marti/api/cot/xml/:uid", getXmlHandler(app))
	f.Get("/Marti/api/sync/metadata/:hash/tool", getMetadataGetHandler(app))
	f.Put("/Marti/api/sync/metadata/:hash/tool", getMetadataPutHandler(app))

	f.Get("/Marti/vcm", getVideoListHandler(app))
	f.Post("/Marti/vcm", getVideoPostHandler(app))

	f.Get("/Marti/api/video", getVideo2ListHandler(app))

	addMissionApi(app, f)
}

func getVersionHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return ctx.SendString(fmt.Sprintf("GoATAK server %s", getVersion()))
	}
}

func getVersionConfigHandler(app *App) fiber.Handler {
	data := make(map[string]any)
	data["api"] = apiVersion
	data["version"] = getVersion()
	data["hostname"] = "0.0.0.0"

	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(makeAnswer("ServerConfig", data))
	}
}

func getEndpointsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		user := app.users.GetUser(username)
		// secAgo := getIntParam(req, "secAgo", 0)
		data := make([]map[string]any, 0)

		app.items.ForEach(func(item *model.Item) bool {
			if user.CanSeeScope(item.GetScope()) && item.GetClass() == model.CONTACT {
				info := make(map[string]any)
				info["uid"] = item.GetUID()
				info["callsign"] = item.GetCallsign()
				info["lastEventTime"] = item.GetLastSeen()

				if item.IsOnline() {
					info["lastStatus"] = "Connected"
				} else {
					info["lastStatus"] = "Disconnected"
				}

				data = append(data, info)
			}

			return true
		})

		return ctx.JSON(makeAnswer("com.bbn.marti.remote.ClientEndpoint", data))
	}
}

func getContactsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		result := make([]*model.Contact, 0)

		app.items.ForEach(func(item *model.Item) bool {
			if user.CanSeeScope(item.GetScope()) && item.GetClass() == model.CONTACT {
				c := &model.Contact{
					UID:      item.GetUID(),
					Callsign: item.GetCallsign(),
					Team:     item.GetMsg().GetTeam(),
					Role:     item.GetMsg().GetRole(),
				}
				result = append(result, c)
			}

			return true
		})

		return ctx.JSON(result)
	}
}

func getMissionQueryHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		user := app.users.GetUser(username)

		hash := ctx.Query("hash")
		if hash == "" {
			return ctx.Status(fiber.StatusNotAcceptable).SendString("no hash")
		}

		c := app.dbm.ResourceQuery().Hash(hash).Scope(user.GetScope()).One()
		if c == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.SendString(resourceUrl(c))
	}
}

func getMissionUploadHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		hash := ctx.Query("hash")
		fname := ctx.Query("filename")

		if hash == "" {
			app.logger.Error("no hash: ")
			return ctx.Status(fiber.StatusNotAcceptable).SendString("no hash" + ctx.Request().URI().QueryArgs().String())
		}

		if fname == "" {
			app.logger.Error("no filename: " + ctx.Request().URI().QueryArgs().String())
			return ctx.Status(fiber.StatusNotAcceptable).SendString("no filename")
		}

		c, err := app.uploadMultipart(ctx, "", hash, fname, true)
		if err != nil {
			app.logger.Error("error", slog.Any("error", err))
			return ctx.SendStatus(fiber.StatusNotAcceptable)
		}

		app.logger.Info(fmt.Sprintf("save packege %s %s %s", c.FileName, c.UID, c.Hash))

		return ctx.SendString(resourceUrl(c))
	}
}

func getUploadHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Query("uid")
		fname := ctx.Query("name")

		if fname == "" {
			app.logger.Error("no name: " + ctx.Request().URI().QueryArgs().String())

			return ctx.Status(fiber.StatusNotAcceptable).SendString("no name")
		}

		switch ctx.Get(fiber.HeaderContentType) {
		case "multipart/form-data":
			c, err := app.uploadMultipart(ctx, uid, "", fname, false)
			if err != nil {
				app.logger.Error("error", slog.Any("error", err))
				return ctx.SendStatus(fiber.StatusNotAcceptable)
			}

			return ctx.SendString(resourceUrl(c))

		default:
			c, err := app.uploadFile(ctx, uid, fname)
			if err != nil {
				app.logger.Error("error", slog.Any("error", err))
				return ctx.SendStatus(fiber.StatusNotAcceptable)
			}

			return ctx.SendString(resourceUrl(c))
		}
	}
}

func (app *App) uploadMultipart(ctx *fiber.Ctx, uid, hash, filename string, pack bool) (*im.Resource, error) {
	username := Username(ctx)
	user := app.users.GetUser(username)

	fh, err := ctx.FormFile("assetfile")

	if err != nil {
		app.logger.Error("error", slog.Any("error", err))
		return nil, err
	}

	f, err := fh.Open()

	if err != nil {
		app.logger.Error("error", slog.Any("error", err))
		return nil, err
	}

	hash1, _, err := app.files.PutFile(user.GetScope(), hash, f)

	if err != nil {
		app.logger.Error("save file error", slog.Any("error", err))
		return nil, err
	}

	if hash != "" && hash != hash1 {
		app.logger.Error("bad hash")
		return nil, err
	}

	c := &im.Resource{
		Scope:          user.GetScope(),
		Hash:           hash1,
		UID:            uid,
		FileName:       filename,
		MIMEType:       fh.Header.Get(fiber.HeaderContentType),
		Size:           int(fh.Size),
		SubmissionUser: user.GetLogin(),
		CreatorUID:     queryIgnoreCase(ctx, "creatorUid"),
		Tool:           "",
		Kw:             tools.NewStringSet(),
	}

	if pack {
		c.Kw.Add("missionpackage")
		c.Tool = "public"
	}

	err = app.dbm.Create(c)

	return c, err
}

func (app *App) uploadFile(ctx *fiber.Ctx, uid, filename string) (*im.Resource, error) {
	username := Username(ctx)
	user := app.users.GetUser(username)

	hash, n, err := app.files.PutFile(user.GetScope(), "", ctx.Context().RequestBodyStream())

	if err != nil {
		app.logger.Error("save file error", slog.Any("error", err))
		return nil, err
	}

	c := &im.Resource{
		Scope:          user.GetScope(),
		Hash:           hash,
		UID:            uid,
		FileName:       filename,
		MIMEType:       ctx.Get(fiber.HeaderContentType),
		Size:           int(n),
		SubmissionUser: user.GetLogin(),
		CreatorUID:     queryIgnoreCase(ctx, "creatorUid"),
		Tool:           "",
		Keywords:       "",
	}

	err = app.dbm.Create(c)

	return c, err
}

func getContentGetHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		user := app.users.GetUser(username)

		hash := ctx.Query("hash")
		uid := ctx.Query("uid")

		if hash == "" && uid == "" {
			return ctx.Status(fiber.StatusNotAcceptable).SendString("no hash or uid")
		}

		fi := app.dbm.ResourceQuery().Scope(user.GetScope()).Hash(hash).UID(uid).One()

		if fi == nil {
			return ctx.Status(fiber.StatusNotFound).SendString("not found")
		}

		f, err := app.files.GetFile(user.GetScope(), hash)

		if err != nil {
			if errors.Is(err, pm.NotFound) {
				app.logger.Info("not found - hash " + hash)

				return ctx.Status(fiber.StatusNotFound).SendString("not found")
			}
			app.logger.Error("get file error", slog.Any("error", err))

			return err
		}

		defer f.Close()

		ctx.Set(fiber.HeaderContentType, fi.MIMEType)
		ctx.Set(fiber.HeaderLastModified, fi.CreatedAt.UTC().Format(http.TimeFormat))
		ctx.Set(fiber.HeaderContentLength, strconv.Itoa(fi.Size))
		ctx.Set("ETag", fi.Hash)

		_, err = io.Copy(ctx.Response().BodyWriter(), f)

		return err
	}
}

func getMetadataGetHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		hash := ctx.Params("hash")
		username := Username(ctx)
		user := app.users.GetUser(username)

		if hash == "" {
			return ctx.Status(fiber.StatusNotAcceptable).SendString("no hash")
		}

		cn := app.dbm.ResourceQuery().Scope(user.GetScope()).Hash(hash).One()

		if cn == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.SendString(cn.Tool)
	}
}

func getMetadataPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		hash := ctx.Params("hash")

		if hash == "" {
			return ctx.Status(fiber.StatusNotAcceptable).SendString("no hash")
		}

		cn := app.dbm.ResourceQuery().Scope(user.GetScope()).Hash(hash).One()

		if cn == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		app.dbm.UpdateContentTool(cn.ID, string(ctx.Body()))

		return nil
	}
}

func getSearchHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		//kw := ctx.Query("keywords")

		user := app.users.GetUser(Username(ctx))

		result := make(map[string]any)

		files := app.dbm.ResourceQuery().Scope(user.GetScope()).Tool(ctx.Query("tool")).Get()
		res := make([]*im.ResourceDTO, len(files))

		for i, f := range files {
			res[i] = im.ToResourceDTO(f)
		}

		result["results"] = res
		result["resultCount"] = len(res)

		return ctx.JSON(result)
	}
}

func getUserRolesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return ctx.JSON([]string{"user", "webuser"})
	}
}

func getAllGroupsHandler(app *App) fiber.Handler {
	g := make(map[string]any)
	g["name"] = "__ANON__"
	g["direction"] = "OUT"
	g["created"] = "2023-01-01"
	g["type"] = "SYSTEM"
	g["bitpos"] = 2
	g["active"] = true

	result := makeAnswer("com.bbn.marti.remote.groups.Group", []map[string]any{g})

	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(result)
	}
}

func getAllGroupsCacheHandler(_ *App) fiber.Handler {
	result := makeAnswer("java.lang.Boolean", true)

	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(result)
	}
}

func getProfileConnectionHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		uid := queryIgnoreCase(ctx, "clientUid")

		if !app.checkUID(uid) {
			return ctx.SendStatus(fiber.StatusForbidden)
		}

		files := app.GetProfileFiles(username, uid)
		if len(files) == 0 {
			return ctx.SendStatus(fiber.StatusNoContent)
		}

		missionPackage := mp.NewMissionPackage("ProfileMissionPackage-"+uuid.New().String(), "Connection")
		missionPackage.Param("onReceiveImport", "true")
		missionPackage.Param("onReceiveDelete", "true")

		for _, f := range files {
			missionPackage.AddFile(f)
		}

		ctx.Set(fiber.HeaderContentType, "application/zip")
		ctx.Set(fiber.HeaderContentDisposition, "attachment; filename=profile.zip")

		dat, err := missionPackage.Create()
		if err != nil {
			return err
		}

		return ctx.Send(dat)
	}
}

func getVideoListHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		r := new(model.VideoConnections)
		user := app.users.GetUser(Username(ctx))

		app.feeds.ForEach(func(f *model.Feed2) bool {
			if user.CanSeeScope(f.Scope) {
				r.Feeds = append(r.Feeds, f.ToFeed())
			}

			return true
		})

		return ctx.XML(r)
	}
}

func getVideo2ListHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		conn := make([]*model.VideoConnections2, 0)
		user := app.users.GetUser(Username(ctx))

		app.feeds.ForEach(func(f *model.Feed2) bool {
			if user.CanSeeScope(f.Scope) {
				conn = append(conn, &model.VideoConnections2{Feeds: []*model.Feed2{f}})
			}

			return true
		})

		r := make(map[string]any)
		r["videoConnections"] = conn

		return ctx.JSON(r)
	}
}

func getVideoPostHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		user := app.users.GetUser(username)

		r := new(model.VideoConnections)

		if err := ctx.BodyParser(r); err != nil {
			return err
		}

		for _, f := range r.Feeds {
			app.feeds.Store(f.ToFeed2().WithUser(username).WithScope(user.Scope))
		}

		return nil
	}
}

func getXmlHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Params("uid")

		if uid == "" {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		var evt *cotproto.CotEvent
		if item := app.items.Get(uid); item != nil {
			evt = item.GetMsg().GetTakMessage().GetCotEvent()
		} else {
			di := app.dbm.GetPoint(uid)
			if di != nil {
				evt = di.GetEvent()
			}
		}

		if evt == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.XML(cot.CotToEvent(evt))
	}
}

func resourceUrl(c *im.Resource) string {
	return fmt.Sprintf("/Marti/sync/content?hash=%s", c.Hash)
}

func makeAnswer(typ string, data any) map[string]any {
	result := make(map[string]any)
	result["version"] = apiVersion
	result["type"] = typ
	result["nodeId"] = nodeID
	result["data"] = data

	return result
}
