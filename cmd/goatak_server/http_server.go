package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aofei/air"
	"github.com/google/uuid"
	"github.com/kdudkov/goatak/model"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

const (
	baseDir      = "./data"
	infoFileName = "info.json"
)

type HttpServer struct {
	app *App
	air *air.Air
}

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

func NewHttp(app *App, address string) *HttpServer {
	a := air.New()
	a.Address = address

	a.FILE("/", "static/index.html")
	a.FILES("/static", "static")

	a.GET("/config", getConfigHandler(app))
	a.GET("/units", getUnitsHandler(app))

	a.GET("/Marti/api/version", getVersionHandler())
	a.GET("/Marti/api/version/config", getVersionConfigHandler())
	a.GET("/Marti/api/clientEndPoints", getEndpointsHandler())

	a.GET("/Marti/sync/search", getSearchHandler(app))
	a.GET("/Marti/sync/missionquery", getMissionQueryHandler(app))
	a.POST("/Marti/sync/missionupload", getMissionUploadyHandler(app))

	a.GET("/Marti/sync/content", getMetadataGetHandler(app))

	a.GET("/Marti/api/sync/metadata/:hash/tool", getMetadataGetHandler(app))
	a.PUT("/Marti/api/sync/metadata/:hash/tool", getMetadataPutHandler(app))

	a.GET("/stack", getStackHandler())

	a.NotFoundHandler = getNotFoundHandler(app)

	a.RendererTemplateLeftDelim = "[["
	a.RendererTemplateRightDelim = "]]"

	return &HttpServer{
		app: app,
		air: a,
	}
}

func (h *HttpServer) Serve() error {
	return h.air.Serve()
}

func getNotFoundHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("404 - %s %s", req.Method, req.Path)
		res.Status = http.StatusNotFound
		return errors.New(http.StatusText(res.Status))
	}
}

func getConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	m := make(map[string]interface{}, 0)
	m["lat"] = app.lat
	m["lon"] = app.lon
	m["zoom"] = app.zoom
	m["version"] = gitRevision
	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON(m)
	}
}

func getUnitsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		units := make([]*model.WebUnit, 0)

		app.units.Range(func(key, value interface{}) bool {
			switch v := value.(type) {
			case *model.Unit:
				units = append(units, v.ToWeb())
			case *model.Contact:
				units = append(units, v.ToWeb())
			}
			return true
		})

		r := make(map[string]interface{}, 0)
		r["units"] = units

		return res.WriteJSON(r)
	}
}

func getStackHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		return pprof.Lookup("goroutine").WriteTo(res.Body, 1)
	}
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

func getEndpointsHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
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
		hash := req.Param("hash").Value().String()
		dir := filepath.Join(baseDir, hash)
		if _, err := os.Lstat(dir); err != nil {
			res.Status = http.StatusNotFound
			return res.WriteString("not found")
		}

		return res.WriteString(fmt.Sprintf("/Marti/api/sync/metadata/%s/tool", hash))
	}
}

func getMissionUploadyHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s", req.RawQuery())
		hash := req.Param("hash").Value().String()
		fname := req.Param("filename").Value().String()
		if hash == "" {
			app.Logger.Errorf("no hash: %s", req.RawQuery())
		}
		if fname == "" {
			app.Logger.Errorf("no filename: %s", req.RawQuery())
		}

		dir := filepath.Join(baseDir, hash)
		finfo := PackageInfo{
			PrimaryKey:         1,
			UID:                uuid.New().String(),
			SubmissionDateTime: time.Now(),
			Hash:               hash,
			Name:               fname,
			CreatorUID:         req.Param("creatorUid").Value().String(),
			SubmissionUser:     "somebody",
			Tool:               "public",
			Keywords:           make([]string, 0),
		}

		if f, fh, err := req.HTTPRequest().FormFile("assetfile"); err == nil {
			if err := os.MkdirAll(dir, 0777); err != nil {
				return err
			}

			n, err := saveFile(dir, fh.Filename, f)
			if err != nil {
				return err
			}

			finfo.Size = n
			finfo.MIMEType = fh.Header.Get("Content-type")

			fn1, _ := os.Create(filepath.Join(dir, infoFileName))
			dat, _ := json.Marshal(finfo)
			fn1.Write(dat)
			fn1.Close()

			app.Logger.Infof("save packege %s %s", fname, hash)
			return res.WriteString(fmt.Sprintf("/Marti/api/sync/metadata/%s/tool", hash))
		} else {
			return err
		}
	}
}

func getMetadataGetHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		hash := req.Param("hash").Value().String()

		if files, err := ioutil.ReadDir(filepath.Join(baseDir, hash)); err == nil {
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
		return res.WriteString("not foind")
	}
}

func getMetadataPutHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		//hash := req.Param("hash").Value().String()
		s, _ := ioutil.ReadAll(req.Body)
		app.Logger.Infof("body: %s", s)

		return nil
	}
}

func getSearchHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		//kw := req.Param("keywords").Value().String()
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
			if dat, err := ioutil.ReadFile(filepath.Join(baseDir, f.Name(), infoFileName)); err == nil {
				pi := &PackageInfo{}
				if err := json.Unmarshal(dat, pi); err == nil {
					if len(pi.Keywords) == 0 {
						pi.Keywords = []string{"missionpackage"}
					}
					if pi.Tool == "" {
						pi.Tool = "public"
					}
					packages = append(packages, pi)
				} else {
					app.Logger.Errorf("error :%v", err)
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
