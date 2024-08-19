package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/kdudkov/goatak/cmd/goatak_server/mp"
	"github.com/kdudkov/goatak/internal/pm"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/log"

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
		f:    fiber.New(fiber.Config{EnablePrintRoutes: false, DisableStartupMessage: true}),
		addr: addr,
	}

	api.f.Use(log.NewFiberLogger("marti_api", Username))

	addMartiRoutes(app, api.f)

	if app.config.useSsl {
		api.tls = true
		api.cert = *app.config.tlsCert
		api.certPool = app.config.certPool

		//api.TLSConfig = &tls.Config{
		//	Certificates: []tls.Certificate{*app.config.tlsCert},
		//	ClientCAs:    app.config.certPool,
		//	RootCAs:      app.config.certPool,
		//	ClientAuth:   tls.RequireAndVerifyClientCert,
		//	MinVersion:   tls.VersionTLS10,
		//}
	}

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

func addMartiRoutes(app *App, f *fiber.App) {
	f.Get("/Marti/api/version", getVersionHandler(app))
	f.Get("/Marti/api/version/config", getVersionConfigHandler(app))
	f.Get("/Marti/api/clientEndPoints", getEndpointsHandler(app))
	f.Get("/Marti/api/contacts/all", getContactsHandler(app))

	f.Get("/Marti/api/cot/xml/:uid", getXmlHandler(app))

	f.Get("/Marti/api/util/user/roles", getUserRolesHandler(app))

	f.Get("/Marti/api/groups/all", getAllGroupsHandler(app))
	f.Get("/Marti/api/groups/groupCacheEnabled", getAllGroupsCacheHandler(app))

	f.Get("/Marti/api/device/profile/connection", getProfileConnectionHandler(app))

	f.Get("/Marti/sync/search", getSearchHandler(app))
	f.Get("/Marti/sync/missionquery", getMissionQueryHandler(app))
	f.Post("/Marti/sync/missionupload", getMissionUploadHandler(app))
	f.Get("/Marti/api/sync/metadata/:hash/tool", getMetadataGetHandler(app))
	f.Put("/Marti/api/sync/metadata/:hash/tool", getMetadataPutHandler(app))

	f.Get("/Marti/sync/content", getContentGetHandler(app))
	f.Post("/Marti/sync/upload", getUploadHandler(app))

	f.Get("/Marti/vcm", getVideoListHandler(app))
	f.Post("/Marti/vcm", getVideoPostHandler(app))

	f.Get("/Marti/api/video", getVideo2ListHandler(app))

	if app.config.dataSync {
		addMissionApi(app, f)
	}
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
			if !user.CanSeeScope(item.GetScope()) {
				return true
			}

			if item.GetClass() == model.CONTACT {
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
			if !user.CanSeeScope(item.GetScope()) {
				return true
			}

			if item.GetClass() == model.CONTACT {
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

		if pi := app.packageManager.GetFirst(func(pi *pm.PackageInfo) bool {
			return pi.Hash == hash && user.CanSeeScope(pi.Scope)
		}); pi != nil {
			return ctx.SendString(packageUrl(pi))
		}

		return ctx.Status(fiber.StatusNotFound).SendString("not found")
	}
}

func getMissionUploadHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		user := app.users.GetUser(username)
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

		if pi := app.packageManager.GetFirst(func(pi *pm.PackageInfo) bool {
			return pi.Hash == hash && user.CanSeeScope(pi.Scope)
		}); pi != nil {
			app.logger.Info("hash already exists: " + hash)
			return ctx.SendString(packageUrl(pi))
		}

		pi, err := app.uploadMultipart(ctx, "", hash, fname, true)
		if err != nil {
			app.logger.Error("error", "error", err)
			return ctx.SendStatus(fiber.StatusNotAcceptable)
		}

		app.logger.Info(fmt.Sprintf("save packege %s %s %s", pi.Name, pi.UID, pi.Hash))

		return ctx.SendString(packageUrl(pi))
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
			pi, err := app.uploadMultipart(ctx, uid, "", fname, false)
			if err != nil {
				app.logger.Error("error", "error", err)
				return ctx.SendStatus(fiber.StatusNotAcceptable)
			}

			return ctx.SendString(fmt.Sprintf("/Marti/sync/content?hash=%s", pi.Hash))

		default:
			pi, err := app.uploadFile(ctx, uid, fname)
			if err != nil {
				app.logger.Error("error", "error", err)
				return ctx.SendStatus(fiber.StatusNotAcceptable)
			}

			return ctx.SendString(fmt.Sprintf("/Marti/sync/content?hash=%s", pi.Hash))
		}
	}
}

func (app *App) uploadMultipart(ctx *fiber.Ctx, uid, hash, filename string, pack bool) (*pm.PackageInfo, error) {
	username := Username(ctx)
	user := app.users.GetUser(username)

	fh, err := ctx.FormFile("assetfile")

	if err != nil {
		app.logger.Error("error", "error", err)
		return nil, err
	}

	pi := &pm.PackageInfo{
		UID:                uid,
		SubmissionDateTime: time.Now(),
		Keywords:           nil,
		MIMEType:           fh.Header.Get("Content-Type"),
		Size:               int(fh.Size),
		SubmissionUser:     user.GetLogin(),
		PrimaryKey:         0,
		Hash:               hash,
		CreatorUID:         getStringParamIgnoreCaps(ctx, "creatorUid"),
		Scope:              user.GetScope(),
		Name:               filename,
		Tool:               "",
	}

	if pack {
		pi.Keywords = []string{"missionpackage"}
		pi.Tool = "public"
	}

	f, err := fh.Open()

	if err != nil {
		app.logger.Error("error", "error", err)
		return nil, err
	}

	if err := app.packageManager.SaveFile(pi, f); err != nil {
		app.logger.Error("save file error", "error", err)
		return nil, err
	}

	return pi, nil
}

func (app *App) uploadFile(ctx *fiber.Ctx, uid, filename string) (*pm.PackageInfo, error) {
	username := Username(ctx)
	user := app.users.GetUser(username)

	pi := &pm.PackageInfo{
		UID:                uid,
		SubmissionDateTime: time.Now(),
		Keywords:           nil,
		MIMEType:           ctx.Get(fiber.HeaderContentType),
		Size:               0,
		SubmissionUser:     user.GetLogin(),
		PrimaryKey:         0,
		Hash:               "",
		CreatorUID:         getStringParamIgnoreCaps(ctx, "creatorUid"),
		Scope:              user.GetScope(),
		Name:               filename,
		Tool:               "",
	}

	if err1 := app.packageManager.SaveFile(pi, ctx.Request().BodyStream()); err1 != nil {
		app.logger.Error("save file error", "error", err1)
		return nil, err1
	}

	return pi, nil
}

func getContentGetHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		user := app.users.GetUser(username)

		if hash := ctx.Query("hash"); hash != "" {
			f, err := app.packageManager.GetFile(hash)

			if err != nil {
				if errors.Is(err, pm.NotFound) {
					app.logger.Info("not found - hash " + hash)

					return ctx.Status(fiber.StatusNotFound).SendString("not found")
				}
				app.logger.Error("get file error", "error", err)

				return err
			}

			defer f.Close()

			ctx.Set("ETag", hash)

			if size, err := app.packageManager.GetFileSize(hash); err == nil {
				ctx.Set(fiber.HeaderContentLength, strconv.Itoa(int(size)))
			}

			_, err = io.Copy(ctx.Response().BodyWriter(), f)

			return err
		}

		if uid := ctx.Query("uid"); uid != "" {
			if pi := app.packageManager.Get(uid); pi != nil && user.CanSeeScope(pi.Scope) {
				f, err := app.packageManager.GetFile(pi.Hash)

				if err != nil {
					app.logger.Error("get file error", "error", err)
					return err
				}

				defer f.Close()

				ctx.Set(fiber.HeaderContentType, pi.MIMEType)
				ctx.Set(fiber.HeaderLastModified, pi.SubmissionDateTime.UTC().Format(http.TimeFormat))
				ctx.Set(fiber.HeaderContentLength, strconv.Itoa(pi.Size))
				ctx.Set(fiber.HeaderETag, pi.Hash)

				_, err = io.Copy(ctx.Response().BodyWriter(), f)

				return err
			}

			app.logger.Info("not found - uid " + uid)

			return ctx.Status(fiber.StatusNotFound).SendString("not found")
		}

		return ctx.Status(fiber.StatusNotAcceptable).SendString("no hash or uid")
	}
}

func getMetadataGetHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		hash := ctx.Query("hash")
		username := Username(ctx)
		user := app.users.GetUser(username)

		if hash == "" {
			return ctx.Status(fiber.StatusNotAcceptable).SendString("no hash")
		}

		if pi := app.packageManager.GetFirst(func(pi *pm.PackageInfo) bool {
			return pi.Hash == hash && user.CanSeeScope(pi.Scope)
		}); pi != nil {
			return ctx.SendString(pi.Tool)
		}

		return ctx.SendStatus(fiber.StatusNotFound)
	}
}

func getMetadataPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		hash := ctx.Query("hash")

		if hash == "" {
			return ctx.Status(fiber.StatusNotAcceptable).SendString("no hash")
		}

		pis := app.packageManager.GetList(func(pi *pm.PackageInfo) bool {
			return pi.Hash == hash && user.CanSeeScope(pi.Scope)
		})

		for _, pi := range pis {
			pi.Tool = string(ctx.Body())
			app.packageManager.Store(pi)
		}

		return nil
	}
}

func getSearchHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		kw := ctx.Query("keywords")
		tool := ctx.Query("tool")

		user := app.users.GetUser(Username(ctx))

		result := make(map[string]any)

		packages := app.packageManager.GetList(func(pi *pm.PackageInfo) bool {
			return user.CanSeeScope(pi.Scope) && pi.HasKeyword(kw) && (tool == "" || pi.Tool == tool)
		})

		result["results"] = packages
		result["resultCount"] = len(packages)

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
		uid := getStringParamIgnoreCaps(ctx, "clientUid")

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
		uid := ctx.Query("uid")

		if uid == "" {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		var evt *cotproto.CotEvent
		if item := app.items.Get(uid); item != nil {
			evt = item.GetMsg().GetTakMessage().GetCotEvent()
		} else {
			di := app.missions.GetPoint(uid)
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

func packageUrl(pi *pm.PackageInfo) string {
	return fmt.Sprintf("/Marti/sync/content?hash=%s", pi.Hash)
}

func makeAnswer(typ string, data any) map[string]any {
	result := make(map[string]any)
	result["version"] = apiVersion
	result["type"] = typ
	result["nodeId"] = nodeID
	result["data"] = data

	return result
}
