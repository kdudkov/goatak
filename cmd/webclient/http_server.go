package main

import (
	"embed"

	"github.com/aofei/air"
	"github.com/kdudkov/goatak/staticfiles"

	"runtime/pprof"

	"github.com/kdudkov/goatak/model"
)

//go:embed templates
var templates embed.FS

func NewHttp(app *App, address string) *air.Air {
	srv := air.New()
	srv.Address = address

	staticfiles.EmbedFiles(srv, "/static")
	renderer := new(staticfiles.Renderer)
	renderer.LeftDelimeter = "[["
	renderer.RightDelimeter = "]]"
	renderer.Load(templates, "templates")

	srv.GET("/", getIndexHandler(app, renderer))
	srv.GET("/units", getUnitsHandler(app))
	srv.GET("/config", getConfigHandler(app))

	srv.GET("/stack", getStackHandler())

	srv.RendererTemplateLeftDelim = "[["
	srv.RendererTemplateRightDelim = "]]"
	return srv
}

func getIndexHandler(app *App, r *staticfiles.Renderer) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		data := map[string]interface{}{
			"js": []string{"map.js"},
		}
		s, err := r.Render(data, "map.html", "header.html")
		if err != nil {
			return err
		}
		return res.WriteHTML(s)
	}
}

func getUnitsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		r := make(map[string]interface{}, 0)
		r["units"] = getUnits(app)
		r["points"] = getPoints(app)

		return res.WriteJSON(r)
	}
}

func getConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	m := make(map[string]interface{}, 0)
	m["version"] = gitRevision + ":" + gitCommit
	m["lat"] = app.lat
	m["lon"] = app.lon
	m["zoom"] = app.zoom
	m["myuid"] = app.uid
	m["callsign"] = app.callsign
	m["team"] = app.team
	m["role"] = app.role
	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON(m)
	}
}

func getStackHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		return pprof.Lookup("goroutine").WriteTo(res.Body, 1)
	}
}

func getUnits(app *App) []*model.WebUnit {
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

	return units
}

func getPoints(app *App) []*model.WebPoint {
	points := make([]*model.WebPoint, 0)

	app.points.Range(func(key, value interface{}) bool {
		points = append(points, (value.(*model.Point)).ToWeb())
		return true
	})

	return points
}
