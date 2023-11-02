package main

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aofei/air"
	"github.com/google/uuid"
	"github.com/kdudkov/goatak/pkg/model"
	"go.uber.org/zap"
)

const nodeId = "1"

func getMartiApi(app *App, addr string) *air.Air {
	api := air.New()
	api.Address = addr

	addMartiRoutes(app, api, "marti")

	api.NotFoundHandler = getNotFoundHandler(app, "marti")

	if app.config.useSsl {
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{*app.config.tlsCert},
			ClientCAs:    app.config.certPool,
			RootCAs:      app.config.certPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS10,
		}

		api.TLSConfig = tlsCfg
		api.Gases = append(api.Gases, SslCheckHandler(app))
	}

	return api
}

func addMartiRoutes(app *App, api *air.Air, name string) {
	api.GET("/Marti/api/version", getVersionHandler(app, name))
	api.GET("/Marti/api/version/config", getVersionConfigHandler(app, name))
	api.GET("/Marti/api/clientEndPoints", getEndpointsHandler(app, name))
	api.GET("/Marti/api/contacts/all", getContactsHandler(app, name))
	api.GET("/Marti/api/sync/metadata/:hash/tool", getMetadataGetHandler(app, name))
	api.PUT("/Marti/api/sync/metadata/:hash/tool", getMetadataPutHandler(app, name))

	api.GET("/Marti/api/util/user/roles", getUserRolesHandler(app, name))

	api.GET("/Marti/api/groups/all", getAllGroupsHandler(app, name))

	api.GET("/Marti/api/device/profile/connection", getProfileConnectionHandler(app, name))

	api.GET("/Marti/api/missions", getMissionsHandler(app, name))
	api.GET("/Marti/api/missions/", getMissionsHandler(app, name))
	api.GET("/Marti/api/missions/:missionname", getMissionHandler(app, name))

	api.GET("/Marti/sync/content", getMetadataGetHandler(app, name))
	api.GET("/Marti/sync/search", getSearchHandler(app, name))
	api.GET("/Marti/sync/missionquery", getMissionQueryHandler(app, name))
	api.POST("/Marti/sync/missionupload", getMissionUploadHandler(app, name))

	api.GET("/Marti/vcm", getVideoListHandler(app, name))
	api.POST("/Marti/vcm", getVideoPostHandler(app, name))

	api.GET("/Marti/api/video", getVideo2ListHandler(app, name))
}

func getVersionHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteString(fmt.Sprintf("GoATAK server %s", getVersion()))
	}
}

func getVersionConfigHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	data := make(map[string]any)
	data["api"] = "3"
	data["version"] = getVersion()
	data["hostname"] = "0.0.0.0"
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteJSON(makeAnswer("ServerConfig", data))
	}
}

func getEndpointsHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		//secAgo := getIntParam(req, "secAgo", 0)

		data := make([]map[string]any, 0)

		app.items.ForEach(func(item *model.Item) bool {
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
		return res.WriteJSON(makeAnswer("com.bbn.marti.remote.ClientEndpoint", data))
	}
}

func getContactsHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)

		result := make([]*model.Contact, 0)

		app.items.ForEach(func(item *model.Item) bool {
			if item.GetClass() == model.CONTACT {
				c := &model.Contact{
					Uid:      item.GetUID(),
					Callsign: item.GetCallsign(),
					Team:     item.GetMsg().GetTeam(),
					Role:     item.GetMsg().GetRole(),
				}
				result = append(result, c)
			}
			return true
		})
		return res.WriteJSON(result)
	}
}

func getMissionQueryHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		hash := getStringParam(req, "hash")
		if hash == "" {
			res.Status = http.StatusNotAcceptable
			return res.WriteString("no hash")
		}
		if _, ok := app.packageManager.Get(hash); ok {
			return res.WriteString(fmt.Sprintf("/Marti/sync/content?hash=%s", hash))
		} else {
			res.Status = http.StatusNotFound
			return res.WriteString("not found")
		}
	}
}

func getMissionUploadHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		hash := getStringParam(req, "hash")
		fname := getStringParam(req, "filename")

		params := []string{}
		for _, r := range req.Params() {
			params = append(params, r.Name+"="+r.Value().String())
		}

		logger.Infof("params: %s", strings.Join(params, ","))

		if hash == "" {
			logger.Errorf("no hash: %s", req.RawQuery())
			res.Status = http.StatusNotAcceptable
			return res.WriteString("no hash")
		}
		if fname == "" {
			logger.Errorf("no filename: %s", req.RawQuery())
			res.Status = http.StatusNotAcceptable
			return res.WriteString("no filename")
		}

		info := &PackageInfo{
			PrimaryKey:         1,
			UID:                uuid.New().String(),
			SubmissionDateTime: time.Now(),
			Hash:               hash,
			Name:               fname,
			CreatorUID:         getStringParam(req, "creatorUid"),
			SubmissionUser:     user,
			Tool:               "public",
			Keywords:           []string{"missionpackage"},
		}

		if f, fh, err := req.HTTPRequest().FormFile("assetfile"); err == nil {
			n, err := app.packageManager.SaveFile(hash, fh.Filename, f)
			if err != nil {
				logger.Errorf("%v", err)
				return err
			}

			info.Size = n
			info.MIMEType = fh.Header.Get("Content-type")

			app.packageManager.Store(hash, info)

			logger.Infof("save packege %s %s", fname, hash)
			return res.WriteString(fmt.Sprintf("/Marti/sync/content?hash=%s", hash))
		} else {
			logger.Errorf("%v", err)
			return err
		}
	}
}

func getMetadataGetHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		hash := getStringParam(req, "hash")

		if hash == "" {
			res.Status = http.StatusNotAcceptable
			return res.WriteString("no hash")
		}

		if pi, ok := app.packageManager.Get(hash); ok {
			res.Header.Set("Content-type", pi.MIMEType)
			return res.WriteFile(app.packageManager.GetFilePath(hash))
		} else {
			logger.Infof("not found - %s", hash)
			res.Status = http.StatusNotFound
			return res.WriteString("not found")
		}
	}
}

func getMetadataPutHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		hash := getStringParam(req, "hash")

		if hash == "" {
			res.Status = http.StatusNotAcceptable
			return res.WriteString("no hash")
		}

		s, _ := io.ReadAll(req.Body)

		if pi, ok := app.packageManager.Get(hash); ok {
			pi.Tool = string(s)
			app.packageManager.Store(hash, pi)
		}
		logger.Debugf("body: %s", s)

		return nil
	}
}

func getSearchHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		kw := getStringParam(req, "keywords")
		tool := getStringParam(req, "tool")

		result := make(map[string]any)
		packages := app.packageManager.GetList(kw, tool)

		result["results"] = packages
		result["resultCount"] = len(packages)
		return res.WriteJSON(result)
	}
}

func getUserRolesHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteJSON([]string{"user", "webuser"})
	}
}

func getAllGroupsHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	g := make(map[string]any)
	g["name"] = "__ANON__"
	g["direction"] = "OUT"
	g["created"] = "2023-01-01"
	g["type"] = "SYSTEM"
	g["bitpos"] = 2
	g["active"] = true

	result := makeAnswer("com.bbn.marti.remote.groups.Group", []map[string]any{g})

	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteJSON(result)
	}
}

func getMissionsHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteJSON(makeAnswer("Mission", []string{}))
	}
}

func getMissionHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)

		m := GetDefault(getStringParam(req, "missionname"))
		return res.WriteJSON(makeAnswer("Mission", []any{m}))
	}
}

func getProfileConnectionHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)
		_ = getIntParam(req, "syncSecago", 0)
		uid := getStringParamIgnoreCaps(req, "clientUid")

		files := app.GetProfileFiles(user, uid)
		if len(files) == 0 {
			res.Status = http.StatusNoContent
			return nil
		}

		mp := NewMissionPackage("ProfileMissionPackage-"+uuid.New().String(), "Connection")
		mp.Param("onReceiveImport", "true")
		mp.Param("onReceiveDelete", "true")
		for _, f := range files {
			mp.AddFile(f)
		}

		res.Header.Set("Content-Type", "application/zip")
		res.Header.Set("Content-Disposition", "attachment; filename=profile.zip")
		dat, err := mp.Create()
		if err != nil {
			return err
		}
		return res.Write(bytes.NewReader(dat))
	}
}

func getVideoListHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)

		r := new(model.VideoConnections)
		r.XMLName = xml.Name{Local: "videoConnections"}
		app.feeds.ForEach(func(f *model.Feed2) bool {
			r.Feeds = append(r.Feeds, f.ToFeed())
			return true
		})
		return res.WriteXML(r)
	}
}

func getVideo2ListHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)

		conn := make([]*model.VideoConnections2, 0)
		app.feeds.ForEach(func(f *model.Feed2) bool {
			conn = append(conn, &model.VideoConnections2{Feeds: []*model.Feed2{f}})
			return true
		})

		r := make(map[string]any)
		r["videoConnections"] = conn
		return res.WriteJSON(r)
	}
}

func getVideoPostHandler(app *App, name string) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		logger := app.Logger.With(zap.String("api", name), zap.String("user", user))
		logger.Infof("%s %s", req.Method, req.Path)

		r := new(model.VideoConnections)

		decoder := xml.NewDecoder(req.Body)
		if err := decoder.Decode(r); err != nil {
			return err
		}
		for _, f := range r.Feeds {
			app.feeds.Store(f.ToFeed2().WithUser(user))
		}
		return nil
	}
}

func makeAnswer(typ string, data any) map[string]any {
	result := make(map[string]any)
	result["version"] = "3"
	result["type"] = typ
	result["nodeId"] = nodeId
	result["data"] = data
	return result
}

func getStringParam(req *air.Request, name string) string {
	p := req.Param(name)
	if p == nil {
		return ""
	}

	return p.Value().String()
}

func getIntParam(req *air.Request, name string, def int) int {
	p := req.Param(name)
	if p == nil {
		return def
	}

	if n, err := p.Value().Int(); err == nil {
		return n
	}

	return def
}

func getStringParamIgnoreCaps(req *air.Request, name string) string {
	nn := strings.ToLower(name)
	for _, p := range req.Params() {
		if strings.ToLower(p.Name) == nn {
			return p.Value().String()
		}
	}

	return ""
}
