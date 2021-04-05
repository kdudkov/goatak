package main

import (
	"encoding/json"
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

type PackageInfo struct {
	UID                string    `json:"UID"`
	SubmissionDateTime time.Time `json:"SubmissionDateTime"`
	Keywords           []string  `json:"Keywords"`
	MIMEType           string    `json:"MIMEType"`
	Size               int64     `json:"Size"`
	SubmissionUser     string    `json:"SubmissionUser"`
	PrimaryKey         int       `json:"PrimaryKey"`
	Hash               string    `json:"Hash"`
	CreatorUID         string    `json:"CreatorUid"`
	Name               string    `json:"Name"`
	Tool               string    `json:"Tool"`
}

func addMartiEndpoints(app *App, a *air.Air) {
	a.GET("/Marti/api/version", getVersionHandler())
	a.GET("/Marti/api/clientEndPoints", getVersionHandler())
	a.GET("/Marti/api/version/config", getVersionConfigHandler())

	a.GET("/Marti/sync/search", getSearchHandler(app))
	a.GET("/Marti/sync/missionquery", getMissionQueryHandler(app))
	a.POST("/Marti/sync/missionupload", getMissionUploadHandler(app))

	a.GET("/Marti/sync/content", getMetadataGetHandler(app))

	a.GET("/Marti/api/sync/metadata/:hash/tool", getMetadataGetHandler(app))
	a.PUT("/Marti/api/sync/metadata/:hash/tool", getMetadataPutHandler(app))
}

func getVersionHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		return res.WriteString(fmt.Sprintf("GoATAK server %s", gitRevision))
	}
}

func getVersionConfigHandler() func(req *air.Request, res *air.Response) error {
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
		dir := filepath.Join(baseDir, hash)
		if !exists(dir) {
			res.Status = http.StatusNotFound
			return res.WriteString("not found")
		}

		return res.WriteString(fmt.Sprintf("/Marti/api/sync/metadata/%s/tool", hash))
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

		dir := filepath.Join(baseDir, hash)
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
			n, err := saveFile(dir, fh.Filename, f)
			if err != nil {
				return err
			}

			info.Size = n
			info.MIMEType = fh.Header.Get("Content-type")

			if err := saveInfo(hash, info); err != nil {
				app.Logger.Errorf("%v", err)
			}

			app.Logger.Infof("save packege %s %s", fname, hash)
			return res.WriteString(fmt.Sprintf("/Marti/api/sync/metadata/%s/tool", hash))
		} else {
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

		dir := filepath.Join(baseDir, hash)

		if !exists(dir) {
			res.Status = http.StatusNotFound
			return res.WriteString("not found")
		}

		if files, err := ioutil.ReadDir(filepath.Join(dir)); err == nil {
			for _, f := range files {
				if f.IsDir() || f.Name() == infoFileName {
					continue
				}
				app.Logger.Infof("get file %s", f.Name())
				return res.WriteFile(filepath.Join(baseDir, hash, f.Name()))
			}
		}

		app.Logger.Infof("not found - %s", hash)
		res.Status = http.StatusNotFound
		return res.WriteString("not found")
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
		app.Logger.Infof("body: %s", s)

		return nil
	}
}

func getSearchHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		kw := getStringParam(req, "keywords")

		//tool := req.Param("tool").Value().String()

		result := make(map[string]interface{}, 0)
		packages := make([]*PackageInfo, 0)

		files, err := ioutil.ReadDir(baseDir)
		if err != nil {
			return err
		}

		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			if pi, err := loadInfo(f.Name()); err == nil {
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

			} else {
				app.Logger.Errorf("error :%v", err)
			}
		}

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

func saveInfo(hash string, finfo *PackageInfo) error {
	fn, err := os.Create(filepath.Join(baseDir, hash, infoFileName))
	if err != nil {
		return err
	}
	defer fn.Close()

	enc := json.NewEncoder(fn)

	return enc.Encode(finfo)
}

func loadInfo(hash string) (*PackageInfo, error) {
	fname := filepath.Join(baseDir, hash, infoFileName)

	if !exists(fname) {
		return nil, fmt.Errorf("info file %s does not exists", fname)
	}

	pi := new(PackageInfo)

	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	err = dec.Decode(pi)

	if err != nil {
		return pi, err
	}
	if pi.Tool == "" {
		pi.Tool = "public"
	}

	return pi, nil
}
