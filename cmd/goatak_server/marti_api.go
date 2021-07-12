package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aofei/air"
	"github.com/google/uuid"
)

func addMartiEndpoints(app *App, a *air.Air) {
	a.GET("/Marti/api/version", getVersionHandler(app))
	a.GET("/Marti/api/version/config", getVersionConfigHandler(app))
	a.GET("/Marti/api/clientEndPoints", getVersionHandler(app))
	a.GET("/Marti/api/sync/metadata/:hash/tool", getMetadataGetHandler(app))
	a.PUT("/Marti/api/sync/metadata/:hash/tool", getMetadataPutHandler(app))

	a.GET("/Marti/sync/content", getMetadataGetHandler(app))
	a.GET("/Marti/sync/search", getSearchHandler(app))
	a.GET("/Marti/sync/missionquery", getMissionQueryHandler(app))
	a.POST("/Marti/sync/missionupload", getMissionUploadHandler(app))
}

func getVersionHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteString(fmt.Sprintf("GoATAK server %s", gitRevision))
	}
}

func getVersionConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	d := make(map[string]interface{}, 0)
	r := make(map[string]interface{}, 0)
	//r["version"] = "3"
	r["type"] = "ServerConfig"
	r["nodeId"] = "1"
	r["data"] = d
	d["api"] = "3"
	d["version"] = gitRevision
	d["hostname"] = "0.0.0.0"
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		return res.WriteJSON(r)
	}
}

func getEndpointsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		d := make(map[string]interface{}, 0)
		r := make(map[string]interface{}, 0)
		r["version"] = "3"
		r["type"] = "com.bbn.marti.remote.ClientEndpoint"
		r["nodeId"] = "1"
		r["data"] = d

		return res.WriteJSON(r)
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
			return res.WriteString(fmt.Sprintf("/Marti/api/sync/metadata/%s/tool", hash))
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
				return err
			}

			info.Size = n
			info.MIMEType = fh.Header.Get("Content-type")

			app.packageManager.Put(hash, info)

			app.Logger.Infof("save packege %s %s", fname, hash)
			return res.WriteString(fmt.Sprintf("/Marti/api/sync/metadata/%s/tool", hash))
		} else {
			return err
		}
	}
}

func getContentGetHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		uid := getStringParam(req, "uid")
		if uid == "" {
			res.Status = http.StatusNotAcceptable
			return res.WriteString("no uid")
		}

		return nil
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
			return res.WriteFile(app.packageManager.GetFile(hash))
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

		s, _ := ioutil.ReadAll(req.Body)

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

		result := make(map[string]interface{}, 0)
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

func saveFile(dir string, fname string, reader io.Reader) (int64, error) {
	if !exists(dir) {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return 0, err
		}
	}

	var n int64
	fn, err := os.Create(filepath.Join(dir, fname))
	if err != nil {
		return 0, err
	}
	if n, err = io.Copy(fn, reader); err != nil {
		return n, err
	}
	if err := fn.Close(); err != nil {
		return n, err
	}

	return n, nil
}

func getStringParam(req *air.Request, name string) string {
	p := req.Param(name)
	if p == nil {
		return ""
	}

	return p.Value().String()
}

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return os.IsExist(err)
	}
	return true
}