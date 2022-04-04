package main

import (
	"embed"
	"encoding/json"
	"runtime/pprof"

	"github.com/aofei/air"
	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/staticfiles"

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
	srv.POST("/dp", getDpHandler(app))
	srv.POST("/pos", getPosHandler(app))

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
		r["messages"] = app.messages
		return res.WriteJSON(r)
	}
}

func getConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		m := make(map[string]interface{}, 0)
		m["version"] = gitRevision
		m["uid"] = app.uid
		lat, lon := app.pos.Get()
		m["lat"] = lat
		m["lon"] = lon
		m["zoom"] = app.zoom
		m["myuid"] = app.uid
		m["callsign"] = app.callsign
		m["team"] = app.team
		m["role"] = app.role

		return res.WriteJSON(m)
	}
}

func getDpHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		dp := new(model.DigitalPointer)
		if req.Body == nil {
			return nil
		}

		if err := json.NewDecoder(req.Body).Decode(dp); err != nil {
			return err
		}

		msg := cot.MakeDpMsg(app.uid, app.typ, app.callsign+"."+dp.Name, dp.Lat, dp.Lon)
		app.AddMsg(msg)
		return res.WriteString("Ok")
	}
}

func getPosHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		pos := make(map[string]float64)
		if req.Body == nil {
			return nil
		}

		if err := json.NewDecoder(req.Body).Decode(&pos); err != nil {
			return err
		}

		lat, latOk := pos["lat"]
		lon, lonOk := pos["lon"]

		if latOk && lonOk {
			app.Logger.Infof("new my coords: %.5f,%.5f", lat, lon)
			app.pos.Set(lat, lon)
		}
		return res.WriteString("Ok")
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
		case *model.Point:
			units = append(units, v.ToWeb())
		}
		return true
	})

	return units
}
