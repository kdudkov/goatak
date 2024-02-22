package main

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aofei/air"
	"github.com/google/uuid"

	"github.com/kdudkov/goatak/internal/pm"

	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/kdudkov/goatak/pkg/model"
)

const (
	nodeID     = "1"
	apiVersion = "3"
)

func getMartiApi(app *App, addr string) *air.Air {
	api := air.New()
	api.Address = addr

	addMartiRoutes(app, api)

	api.NotFoundHandler = getNotFoundHandler()
	api.Gases = append(api.Gases, LoggerGas(app.Logger, "marti_api"))

	if app.config.useSsl {
		api.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{*app.config.tlsCert},
			ClientCAs:    app.config.certPool,
			RootCAs:      app.config.certPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS10,
		}

		api.Gases = append(api.Gases, SSLCheckHandlerGas(app))
	}

	return api
}

func addMartiRoutes(app *App, api *air.Air) {
	api.GET("/Marti/api/version", getVersionHandler(app))
	api.GET("/Marti/api/version/config", getVersionConfigHandler(app))
	api.GET("/Marti/api/clientEndPoints", getEndpointsHandler(app))
	api.GET("/Marti/api/contacts/all", getContactsHandler(app))

	api.GET("/Marti/api/cot/xml/:uid", getXmlHandler(app))

	api.GET("/Marti/api/util/user/roles", getUserRolesHandler(app))

	api.GET("/Marti/api/groups/all", getAllGroupsHandler(app))
	api.GET("/Marti/api/groups/groupCacheEnabled", getAllGroupsCacheHandler(app))

	api.GET("/Marti/api/device/profile/connection", getProfileConnectionHandler(app))

	api.GET("/Marti/sync/content", getContentGetHandler(app))
	api.GET("/Marti/sync/search", getSearchHandler(app))
	api.GET("/Marti/sync/missionquery", getMissionQueryHandler(app))
	api.POST("/Marti/sync/missionupload", getMissionUploadHandler(app))
	api.GET("/Marti/api/sync/metadata/:hash/tool", getMetadataGetHandler(app))
	api.PUT("/Marti/api/sync/metadata/:hash/tool", getMetadataPutHandler(app))
	api.POST("/Marti/sync/upload", getUploadHandler(app))

	api.GET("/Marti/vcm", getVideoListHandler(app))
	api.POST("/Marti/vcm", getVideoPostHandler(app))

	api.GET("/Marti/api/video", getVideo2ListHandler(app))

	if app.config.dataSync {
		addMissionApi(app, api)
	}
}

func getVersionHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		return res.WriteString(fmt.Sprintf("GoATAK server %s", getVersion()))
	}
}

func getVersionConfigHandler(app *App) air.Handler {
	data := make(map[string]any)
	data["api"] = apiVersion
	data["version"] = getVersion()
	data["hostname"] = "0.0.0.0"

	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON(makeAnswer("ServerConfig", data))
	}
}

func getEndpointsHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		// secAgo := getIntParam(req, "secAgo", 0)
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

func getContactsHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		result := make([]*model.Contact, 0)

		app.items.ForEach(func(item *model.Item) bool {
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

		return res.WriteJSON(result)
	}
}

func getMissionQueryHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		username := getUsernameFromReq(req)
		user := app.users.GetUser(username)

		hash := getStringParam(req, "hash")
		if hash == "" {
			res.Status = http.StatusNotAcceptable

			return res.WriteString("no hash")
		}

		if p := app.packageManager.GetByHash(hash); p != nil && user.CanSeeScope(p.Scope) {
			return res.WriteString(fmt.Sprintf("/Marti/sync/content?hash=%s", hash))
		}
		res.Status = http.StatusNotFound

		return res.WriteString("not found")
	}
}

func getMissionUploadHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		hash := getStringParam(req, "hash")
		fname := getStringParam(req, "filename")

		params := []string{}
		for _, r := range req.Params() {
			params = append(params, r.Name+"="+r.Value().String())
		}

		app.Logger.Infof("params: %s", strings.Join(params, ","))

		if hash == "" {
			app.Logger.Errorf("no hash: %s", req.RawQuery())
			res.Status = http.StatusNotAcceptable

			return res.WriteString("no hash")
		}

		if fname == "" {
			app.Logger.Errorf("no filename: %s", req.RawQuery())
			res.Status = http.StatusNotAcceptable

			return res.WriteString("no filename")
		}

		uid := uuid.NewString()

		_, err := app.uploadMultipart(req, uid, hash, fname, true)
		if err != nil {
			app.Logger.Errorf("error: %v", err)
			res.Status = http.StatusNotAcceptable
			return nil
		}

		app.Logger.Infof("save packege %s %s %s", fname, uid, hash)

		return res.WriteString(fmt.Sprintf("/Marti/sync/content?hash=%s", hash))
	}
}

func getUploadHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		uid := getStringParam(req, "uid")
		fname := getStringParam(req, "name")

		if fname == "" {
			app.Logger.Errorf("no name: %s", req.RawQuery())
			res.Status = http.StatusNotAcceptable

			return res.WriteString("no name")
		}

		switch req.Header.Get("Content-type") {
		case "multipart/form-data":
			pi, err := app.uploadMultipart(req, uid, "", fname, false)
			if err != nil {
				app.Logger.Errorf("error: %v", err)
				res.Status = http.StatusNotAcceptable
				return nil
			}

			return res.WriteString(fmt.Sprintf("/Marti/sync/content?hash=%s", pi.Hash))

		default:
			pi, err := app.uploadFile(req, uid, fname)
			if err != nil {
				app.Logger.Errorf("error: %v", err)
				res.Status = http.StatusNotAcceptable
				return nil
			}

			return res.WriteString(fmt.Sprintf("/Marti/sync/content?hash=%s", pi.Hash))
		}
	}
}

func (app *App) uploadMultipart(req *air.Request, uid, hash, filename string, pack bool) (*pm.PackageInfo, error) {
	username := getUsernameFromReq(req)
	user := app.users.GetUser(username)

	f, fh, err := req.HTTPRequest().FormFile("assetfile")

	if err != nil {
		app.Logger.Errorf("error: %v", err)
		return nil, err
	}

	data, err := io.ReadAll(f)

	if err != nil {
		app.Logger.Errorf("error: %v", err)
		return nil, err
	}

	pi, err := app.packageManager.SaveData(uid, hash, filename, fh.Header.Get("Content-type"), data, func(pi1 *pm.PackageInfo) {
		pi1.SubmissionUser = user.GetLogin()
		pi1.Scope = user.GetScope()
		pi1.CreatorUID = getStringParamIgnoreCaps(req, "creatorUid")

		if pack {
			pi1.Keywords = []string{"missionpackage"}
			pi1.Tool = "public"
		}
	})

	if err != nil {
		app.Logger.Errorf("error: %v", err)
		return nil, err
	}

	return pi, nil
}

func (app *App) uploadFile(req *air.Request, uid, filename string) (*pm.PackageInfo, error) {
	username := getUsernameFromReq(req)
	user := app.users.GetUser(username)

	if req.Body == nil {
		return nil, errors.New("no body")
	}

	defer req.Body.Close()

	data, err := io.ReadAll(req.Body)

	if err != nil {
		app.Logger.Errorf("error: %v", err)
		return nil, err
	}

	pi, err := app.packageManager.SaveData(uid, "", filename, req.Header.Get("Content-type"), data, func(pi1 *pm.PackageInfo) {
		pi1.SubmissionUser = user.GetLogin()
		pi1.Scope = user.GetScope()
		pi1.CreatorUID = getStringParamIgnoreCaps(req, "creatorUid")
	})

	if err != nil {
		app.Logger.Errorf("error: %v", err)
		return nil, err
	}

	return pi, nil
}

func getContentGetHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		username := getUsernameFromReq(req)
		user := app.users.GetUser(username)
		
		if hash := getStringParam(req, "hash"); hash != "" {
			if pi := app.packageManager.GetByHash(hash); pi != nil && user.CanSeeScope(pi.Scope) {
				res.Header.Set("Content-type", pi.MIMEType)

				return res.WriteFile(app.packageManager.GetFilePath(pi))
			}
			app.Logger.Infof("not found - hash %s", hash)

			res.Status = http.StatusNotFound

			return res.WriteString("not found")
		}

		if uid := getStringParam(req, "uid"); uid != "" {
			if pi := app.packageManager.Get(uid); pi != nil && user.CanSeeScope(pi.Scope) {
				res.Header.Set("Content-type", pi.MIMEType)

				return res.WriteFile(app.packageManager.GetFilePath(pi))
			}
			app.Logger.Infof("not found - uid %s", uid)

			res.Status = http.StatusNotFound

			return res.WriteString("not found")
		}

		res.Status = http.StatusNotAcceptable

		return res.WriteString("no hash or uid")
	}
}

func getMetadataGetHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		hash := getStringParam(req, "hash")
		username := getUsernameFromReq(req)
		user := app.users.GetUser(username)

		if hash == "" {
			res.Status = http.StatusNotAcceptable

			return res.WriteString("no hash")
		}

		if pi := app.packageManager.GetByHash(hash); pi != nil && user.CanSeeScope(pi.Scope) {
			return res.WriteString(pi.Tool)
		}

		res.Status = http.StatusNotFound

		return nil
	}
}

func getMetadataPutHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		user := app.users.GetUser(getUsernameFromReq(req))
		hash := getStringParam(req, "hash")

		if hash == "" {
			res.Status = http.StatusNotAcceptable

			return res.WriteString("no hash")
		}

		s, _ := io.ReadAll(req.Body)

		if pi := app.packageManager.GetByHash(hash); pi != nil && user.CanSeeScope(pi.Scope)  {
			pi.Tool = string(s)
			app.packageManager.Store(pi.UID, pi)
		}

		return nil
	}
}

func getSearchHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		kw := getStringParam(req, "keywords")
		tool := getStringParam(req, "tool")

		user := app.users.GetUser(getUsernameFromReq(req))

		result := make(map[string]any)
		packages := make([]*pm.PackageInfo, 0)

		for _, pi := range app.packageManager.GetList(kw, tool) {
			if user.CanSeeScope(pi.Scope) {
				packages = append(packages, pi)
			}
		}		

		result["results"] = packages
		result["resultCount"] = len(packages)

		return res.WriteJSON(result)
	}
}

func getUserRolesHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON([]string{"user", "webuser"})
	}
}

func getAllGroupsHandler(app *App) air.Handler {
	g := make(map[string]any)
	g["name"] = "__ANON__"
	g["direction"] = "OUT"
	g["created"] = "2023-01-01"
	g["type"] = "SYSTEM"
	g["bitpos"] = 2
	g["active"] = true

	result := makeAnswer("com.bbn.marti.remote.groups.Group", []map[string]any{g})

	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON(result)
	}
}

func getAllGroupsCacheHandler(_ *App) air.Handler {
	result := makeAnswer("java.lang.Boolean", true)

	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON(result)
	}
}

func getProfileConnectionHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		username := getUsernameFromReq(req)
		_ = getIntParam(req, "syncSecago", 0)
		uid := getStringParamIgnoreCaps(req, "clientUid")

		files := app.GetProfileFiles(username, uid)
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

func getVideoListHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		r := new(model.VideoConnections)

		app.feeds.ForEach(func(f *model.Feed2) bool {
			r.Feeds = append(r.Feeds, f.ToFeed())

			return true
		})

		return res.WriteXML(r)
	}
}

func getVideo2ListHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
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

func getVideoPostHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		username := getUsernameFromReq(req)

		r := new(model.VideoConnections)

		decoder := xml.NewDecoder(req.Body)
		if err := decoder.Decode(r); err != nil {
			return err
		}

		for _, f := range r.Feeds {
			app.feeds.Store(f.ToFeed2().WithUser(username))
		}

		return nil
	}
}

func getXmlHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		var evt *cotproto.CotEvent

		if item := app.items.Get(getStringParam(req, "uid")); item != nil {
			evt = item.GetMsg().GetTakMessage().GetCotEvent()
		} else {
			di := app.missions.GetPoint(getStringParam(req, "uid"))
			if di != nil {
				evt = di.GetEvent()
			}
		}

		if evt == nil {
			res.Status = http.StatusNotFound

			return nil
		}

		return res.WriteXML(evt)
	}
}

func makeAnswer(typ string, data any) map[string]any {
	result := make(map[string]any)
	result["version"] = apiVersion
	result["type"] = typ
	result["nodeId"] = nodeID
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

func getBoolParam(req *air.Request, name string, def bool) bool {
	p := req.Param(name)
	if p == nil {
		return def
	}

	v, _ := p.Value().Bool()
	return v
}

func getStringParams(req *air.Request, name string) []string {
	p := req.Param(name)
	if p == nil {
		return nil
	}

	result := make([]string, len(p.Values))

	for i, v := range p.Values {
		result[i] = v.String()
	}

	return result
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

