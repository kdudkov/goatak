package main

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aofei/air"
	"github.com/google/uuid"
	"github.com/kdudkov/goatak/model"
)

func getMartiApi(app *App, addr string) *air.Air {
	api := air.New()
	api.Address = addr

	addMartiRoutes(app, api)

	api.NotFoundHandler = getNotFoundHandler(app)

	if app.config.useSsl {
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{*app.config.tlsCert},
			ClientCAs:    app.config.certPool,
			RootCAs:      app.config.certPool,
			ClientAuth:   tls.NoClientCert,
			MinVersion:   tls.VersionTLS10,
		}

		api.TLSConfig = tlsCfg
	}

	return api
}

func addMartiRoutes(app *App, api *air.Air) {
	api.GET("/Marti/api/version", getVersionHandler(app))
	api.GET("/Marti/api/version/config", getVersionConfigHandler(app))
	api.GET("/Marti/api/clientEndPoints", getEndpointsHandler(app))
	api.GET("/Marti/api/contacts/all", getContactsHandler(app))
	api.GET("/Marti/api/sync/metadata/:hash/tool", getMetadataGetHandler(app))
	api.PUT("/Marti/api/sync/metadata/:hash/tool", getMetadataPutHandler(app))

	api.GET("/Marti/api/util/user/roles", getUserRolesHandler(app))

	api.GET("/Marti/api/device/profile/connection", getProfileConnectionHandler(app))
	api.GET("/Marti/sync/content", getMetadataGetHandler(app))
	api.GET("/Marti/sync/search", getSearchHandler(app))
	api.GET("/Marti/sync/missionquery", getMissionQueryHandler(app))
	api.POST("/Marti/sync/missionupload", getMissionUploadHandler(app))

	api.GET("/Marti/vcm", getVideoListHandler(app))
	api.POST("/Marti/vcm", getVideoPostHandler(app))
}

func getVersionHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteString(fmt.Sprintf("GoATAK server %s", gitRevision))
	}
}

func getVersionConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	result := make(map[string]any, 0)
	data := make(map[string]any, 0)
	result["version"] = "2"
	result["type"] = "ServerConfig"
	result["nodeId"] = "1"
	data["api"] = "2"
	data["version"] = gitRevision + ":" + gitBranch
	data["hostname"] = "0.0.0.0"
	result["data"] = data
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteJSON(result)
	}
}

func getEndpointsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		//secAgo := getIntParam(req, "secAgo", 0)

		result := make(map[string]any)
		data := make([]map[string]any, 0)
		result["Matcher"] = "com.bbn.marti.remote.ClientEndpoint"
		result["BaseUrl"] = ""
		result["ServerConnectString"] = ""
		result["NotificationId"] = ""
		result["type"] = "com.bbn.marti.remote.ClientEndpoint"

		app.units.Range(func(key, value any) bool {
			c := value.(*model.Item)
			if c.GetClass() == model.CONTACT {
				info := make(map[string]any)
				info["uid"] = c.GetUID()
				info["callsign"] = c.GetCallsign()
				info["lastEventTime"] = c.GetLastSeen()
				if c.IsOnline() {
					info["lastStatus"] = "Connected"
				} else {
					info["lastStatus"] = "Disconnected"
				}
				data = append(data, info)
			}

			return true
		})
		result["data"] = data
		return res.WriteJSON(result)
	}
}

func getContactsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)

		result := make([]map[string]any, 0)

		app.units.Range(func(key, value any) bool {
			c := value.(*model.Item)
			if c.GetClass() == model.CONTACT {
				info := make(map[string]any)
				info["uid"] = c.GetUID()
				info["callsign"] = c.GetCallsign()
				info["team"] = c.GetMsg().GetTeam()
				info["role"] = c.GetMsg().GetRole()
				info["takv"] = ""
				info["notes"] = ""
				info["filterGroups"] = ""
				result = append(result, info)
			}

			return true
		})
		return res.WriteJSON(result)
	}
}

func getMissionQueryHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
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

func getMissionUploadHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		hash := getStringParam(req, "hash")
		fname := getStringParam(req, "filename")

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

		info := &PackageInfo{
			PrimaryKey:         1,
			UID:                uuid.New().String(),
			SubmissionDateTime: time.Now(),
			Hash:               hash,
			Name:               fname,
			CreatorUID:         getStringParam(req, "creatorUid"),
			SubmissionUser:     "somebody",
			Tool:               "public",
			Keywords:           []string{"missionpackage"},
		}

		if f, fh, err := req.HTTPRequest().FormFile("assetfile"); err == nil {
			n, err := app.packageManager.SaveFile(hash, fh.Filename, f)
			if err != nil {
				app.Logger.Errorf("%v", err)
				return err
			}

			info.Size = n
			info.MIMEType = fh.Header.Get("Content-type")

			app.packageManager.Put(hash, info)

			app.Logger.Infof("save packege %s %s", fname, hash)
			return res.WriteString(fmt.Sprintf("/Marti/sync/content?hash=%s", hash))
		} else {
			app.Logger.Errorf("%v", err)
			return err
		}
	}
}

func getMetadataGetHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		hash := getStringParam(req, "hash")

		if hash == "" {
			res.Status = http.StatusNotAcceptable
			return res.WriteString("no hash")
		}

		if pi, ok := app.packageManager.Get(hash); ok {
			res.Header.Set("Content-type", pi.MIMEType)
			return res.WriteFile(app.packageManager.GetFilePath(hash))
		} else {
			app.Logger.Infof("not found - %s", hash)
			res.Status = http.StatusNotFound
			return res.WriteString("not found")
		}
	}
}

func getMetadataPutHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		hash := getStringParam(req, "hash")

		if hash == "" {
			res.Status = http.StatusNotAcceptable
			return res.WriteString("no hash")
		}

		s, _ := io.ReadAll(req.Body)

		if pi, ok := app.packageManager.Get(hash); ok {
			pi.Tool = string(s)
			app.packageManager.Put(hash, pi)
		}
		app.Logger.Infof("body: %s", s)

		return nil
	}
}

func getSearchHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		kw := getStringParam(req, "keywords")

		//tool := getStringParam(req, "tool")

		result := make(map[string]any, 0)
		packages := make([]*PackageInfo, 0)

		app.packageManager.Range(func(key string, pi *PackageInfo) bool {
			if kw != "" {
				for _, s := range pi.Keywords {
					if s == kw {
						packages = append(packages, pi)
						break
					}
				}
			} else {
				packages = append(packages, pi)
			}
			return true
		})

		result["results"] = packages
		result["resultCount"] = len(packages)
		return res.WriteJSON(result)
	}
}

func getUserRolesHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteJSON([]string{"user", "webuser"})
	}
}

func getProfileConnectionHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		_ = getIntParam(req, "syncSecago", 0)
		uid := getStringParamIgnoreCaps(req, "clientUid")

		files := app.userManager.GetProfile("", uid)
		if len(files) == 0 {
			res.Status = http.StatusNoContent
			return nil
		}

		mp := NewMissionPackage("ProfileMissionPackage-"+uuid.New().String(), "Connection")
		mp.Param("onReceiveImport", "true")
		mp.Param("onReceiveDelete", "true")

		for i, f := range files {
			f.SetName(fmt.Sprintf("file%d/%s", i, f.Name()))
			mp.AddFile(f)
		}

		res.Header.Set("Content-Disposition", "attachment; filename=profile.zip")
		dat, err := mp.Create()
		if err != nil {
			return err
		}
		res.Write(bytes.NewReader(dat))
		return nil
	}
}

func getVideoListHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)

		r := new(VideoConnections)
		r.XMLName = xml.Name{Local: "videoConnections"}
		app.feeds.Range(func(_, value any) bool {
			if feed, ok := value.(*Feed); ok {
				r.Feeds = append(r.Feeds, feed)
			}
			return true
		})
		return res.WriteXML(r)
	}
}

func getVideoPostHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)

		r := new(VideoConnections)

		decoder := xml.NewDecoder(req.Body)
		if err := decoder.Decode(r); err != nil {
			return err
		}
		for _, f := range r.Feeds {
			app.feeds.Store(f.Uid, f)
		}
		return nil
	}
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

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return os.IsExist(err)
	}
	return true
}
