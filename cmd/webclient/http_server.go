package main

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/aofei/air"
	"github.com/google/uuid"
	"runtime/pprof"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
	"github.com/kdudkov/goatak/staticfiles"
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
	srv.GET("/config", getConfigHandler(app))
	srv.GET("/types", getTypes)
	srv.POST("/dp", getDpHandler(app))
	srv.POST("/pos", getPosHandler(app))

	srv.GET("/ws", getWsHandler(app))

	srv.GET("/unit", getUnitsHandler(app))
	srv.POST("/unit", addItemHandler(app))
	srv.POST("/message", addMessageHandler(app))
	srv.DELETE("/unit/:uid", deleteItemHandler(app))

	srv.GET("/stack", getStackHandler())

	srv.RendererTemplateLeftDelim = "[["
	srv.RendererTemplateRightDelim = "]]"
	return srv
}

func getIndexHandler(app *App, r *staticfiles.Renderer) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		data := map[string]any{
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
		r := make(map[string]any, 0)
		r["units"] = getUnits(app)
		r["messages"] = app.messages.Chats
		return res.WriteJSON(r)
	}
}

func getConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		m := make(map[string]any, 0)
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
		app.SendMsg(msg)
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

		app.SendMsg(app.MakeMe())
		return res.WriteString("Ok")
	}
}

func addItemHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		wu := new(model.WebUnit)
		if req.Body == nil {
			return nil
		}

		if err := json.NewDecoder(req.Body).Decode(wu); err != nil {
			return err
		}

		if wu == nil {
			return fmt.Errorf("no item")
		}

		msg := wu.ToMsg()

		if wu.Send {
			app.SendMsg(msg.TakMessage)
		}

		if wu.Category == "unit" || wu.Category == "point" {
			if u := app.items.Get(msg.GetUid()); u != nil {
				u.UpdateFromWeb(wu, msg)
			} else {
				app.items.Store(model.FromMsgLocal(msg, wu.Send))
			}

			//app.ProcessItem(msg)
		}

		r := make(map[string]any, 0)
		r["units"] = getUnits(app)
		r["messages"] = app.messages
		return res.WriteJSON(r)
	}
}

func addMessageHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		msg := new(model.ChatMessage)
		if req.Body == nil {
			return nil
		}

		if err := json.NewDecoder(req.Body).Decode(msg); err != nil {
			return err
		}

		if msg == nil {
			return fmt.Errorf("no message")
		}

		if msg.Id == "" {
			msg.Id = uuid.New().String()
		}
		app.SendMsg(model.MakeChatMessage(msg))
		app.messages.Add(msg)
		return res.WriteJSON(map[string]string{"ok": "ok"})
	}
}

func getWsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}

		defer ws.Close()

		name := uuid.New().String()

		ch := make(chan *model.WebUnit)

		go func() {
			for ci := range ch {
				if ws.Closed {
					return
				}

				if b, err := json.Marshal(ci); err == nil {
					if err := ws.WriteText(string(b)); err != nil {
						ws.Close()
						return
					}
				} else {
					ws.Close()
					return
				}
			}
		}()

		app.listeners.Store(name, func(u *model.WebUnit) {
			if ws.Closed {
				return
			}

			select {
			case ch <- u:
			default:
			}
		})

		app.Logger.Debug("ws listener connected")

		ws.BinaryHandler = func(b []byte) error {
			return nil
		}

		ws.Listen()
		app.Logger.Debug("ws listener disconnected")
		app.listeners.Delete(name)
		ws.Close()
		close(ch)

		return nil
	}
}

func deleteItemHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		uid := getStringParam(req, "uid")
		app.items.Remove(uid)

		r := make(map[string]any, 0)
		r["units"] = getUnits(app)
		r["messages"] = app.messages
		return res.WriteJSON(r)
	}
}

func getStackHandler() func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		return pprof.Lookup("goroutine").WriteTo(res.Body, 1)
	}
}

func getUnits(app *App) []*model.WebUnit {
	units := make([]*model.WebUnit, 0)

	app.items.ForEach(func(item *model.Item) bool {
		units = append(units, item.ToWeb())
		return true
	})

	return units
}

func getStringParam(req *air.Request, name string) string {
	p := req.Param(name)
	if p == nil {
		return ""
	}

	return p.Value().String()
}

func getTypes(req *air.Request, res *air.Response) error {
	return res.WriteJSON(cot.Root)
}
