package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"runtime/pprof"

	"github.com/aofei/air"
	"github.com/kdudkov/goatak/model"
	"github.com/kdudkov/goatak/staticfiles"
)

type HttpServer struct {
	app *App
	air *air.Air
}

func NewHttp(app *App, address string) *HttpServer {
	a := air.New()
	a.Address = address

	staticfiles.EmbedFile(a, "/", "static/index.html")
	staticfiles.EmbedFile(a, "/map", "static/map.html")

	staticfiles.EmbedFiles(a, "/static")

	a.GET("/config", getConfigHandler(app))
	a.GET("/units", getUnitsHandler(app))

	a.GET("/stack", getStackHandler())

	addMartiEndpoints(app, a)

	a.NotFoundHandler = getNotFoundHandler(app)

	a.RendererTemplateLeftDelim = "[["
	a.RendererTemplateRightDelim = "]]"

	return &HttpServer{
		app: app,
		air: a,
	}
}

func (h *HttpServer) Serve() error {
	h.app.Logger.Infof("listening http at %s", h.air.Address)
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

func getStaticHandler(efs embed.FS) func(req *air.Request, res *air.Response) error {
	hfs := http.FS(efs)
	fsh := http.FileServer(http.FS(efs))

	return func(req *air.Request, res *air.Response) error {
		fsh.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())
		path := req.Param("*").Value().String()
		fmt.Println(path)
		path = filepath.FromSlash("/" + path)
		path = filepath.Clean(path)

		f, err := hfs.Open(path)
		if err != nil {
			if _, ok := err.(*fs.PathError); ok {
				res.Status = http.StatusNotFound
				return fmt.Errorf("%s is not found", path)
			}
			return err
		}

		if res.Header.Get("Content-Type") == "" {
			res.Header.Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
		}

		return res.Write(f)
	}
}
