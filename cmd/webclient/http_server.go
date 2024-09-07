package main

import (
	"embed"
	"fmt"
	"net/http"
	"runtime/pprof"
	"time"

	"github.com/kdudkov/goatak/internal/wshandler"
	"github.com/kdudkov/goatak/pkg/log"
	"github.com/kdudkov/goatak/staticfiles"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/google/uuid"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
)

//go:embed templates
var templates embed.FS

func NewHttp(app *App) *fiber.App {
	engine := html.NewFileSystem(http.FS(templates), ".html")

	engine.Delims("[[", "]]")

	srv := fiber.New(fiber.Config{EnablePrintRoutes: false, DisableStartupMessage: true, Views: engine})

	srv.Use(log.NewFiberLogger(nil))
	staticfiles.Embed(srv)

	srv.Get("/", getIndexHandler(app))
	srv.Get("/config", getConfigHandler(app))
	srv.Get("/types", getTypes)
	srv.Post("/dp", getDpHandler(app))
	srv.Post("/pos", getPosHandler(app))

	srv.Get("/ws", getWsHandler(app))

	srv.Get("/unit", getUnitsHandler(app))
	srv.Post("/unit", addItemHandler(app))
	srv.Get("/message", getMessagesHandler(app))
	srv.Post("/message", addMessageHandler(app))
	srv.Delete("/unit/:uid", deleteItemHandler(app))

	srv.Get("/stack", getStackHandler())

	return srv
}

func getIndexHandler(_ *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		data := fiber.Map{
			"js": []string{"util.js", "map.js"},
		}

		return ctx.Render("templates/map", data, "templates/header")
	}
}

func getUnitsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(getUnits(app))
	}
}

func getMessagesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return ctx.JSON(app.chatMessages.Chats)
	}
}

func getConfigHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		m := make(map[string]any, 0)
		m["version"] = getVersion()
		m["uid"] = app.uid
		lat, lon := app.pos.Load().GetCoord()
		m["lat"] = lat
		m["lon"] = lon
		m["zoom"] = app.zoom
		m["myuid"] = app.uid
		m["callsign"] = app.callsign
		m["team"] = app.team
		m["role"] = app.role

		m["layers"] = getLayers()

		return ctx.JSON(m)
	}
}

func getDpHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		dp := new(model.DigitalPointer)

		if err := ctx.BodyParser(dp); err != nil {
			return err
		}

		msg := cot.MakeDpMsg(app.uid, app.typ, app.callsign+"."+dp.Name, dp.Lat, dp.Lon)
		app.SendMsg(msg)

		return ctx.SendString("Ok")
	}
}

func getPosHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		pos := make(map[string]float64)

		if err := ctx.BodyParser(&pos); err != nil {
			return err
		}

		lat, latOk := pos["lat"]
		lon, lonOk := pos["lon"]

		if latOk && lonOk {
			app.logger.Info(fmt.Sprintf("new my coords: %.5f,%.5f", lat, lon))
			app.pos.Store(model.NewPos(lat, lon))
		}

		app.SendMsg(app.MakeMe())

		return ctx.SendString("Ok")
	}
}

func addItemHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		wu := new(model.WebUnit)

		if err := ctx.BodyParser(wu); err != nil {
			return err
		}

		msg := wu.ToMsg()

		if wu.Send {
			app.SendMsg(msg.GetTakMessage())
		}

		var u *model.Item
		if wu.Category == "unit" || wu.Category == "point" {
			if u = app.items.Get(msg.GetUID()); u != nil {
				u.Update(msg)
				u.SetSend(wu.Send)
				app.items.Store(u)
			} else {
				u = model.FromMsg(msg)
				u.SetLocal(true)
				u.SetSend(wu.Send)
				app.items.Store(u)
			}
		}

		return ctx.JSON(u.ToWeb())
	}
}

func addMessageHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		msg := new(model.ChatMessage)

		if err := ctx.BodyParser(msg); err != nil {
			return err
		}

		if msg.ID == "" {
			msg.ID = uuid.NewString()
		}

		if msg.Time.IsZero() {
			msg.Time = time.Now()
		}

		if msg.Chatroom != msg.ToUID {
			msg.Direct = true
		}

		m := model.MakeChatMessage(msg)

		app.logger.Debug(m.String())
		app.SendMsg(m)
		app.chatMessages.Add(msg)

		return ctx.JSON(app.chatMessages.Chats)
	}
}

func deleteItemHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Params("uid")
		app.items.Remove(uid)

		r := make(map[string]any, 0)
		r["units"] = getUnits(app)
		r["messages"] = app.chatMessages

		return ctx.JSON(r)
	}
}

func getStackHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return pprof.Lookup("goroutine").WriteTo(ctx.Response().BodyWriter(), 1)
	}
}

func getWsHandler(app *App) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		name := uuid.NewString()

		h := wshandler.NewHandler(app.logger, name, c)

		app.changeCb.SubscribeNamed(name, h.SendItem)
		app.deleteCb.SubscribeNamed(name, h.DeleteItem)
		app.chatCb.SubscribeNamed(name, h.NewChatMessage)
		h.Listen()
	})
}

func getUnits(app *App) []*model.WebUnit {
	units := make([]*model.WebUnit, 0)

	app.items.ForEach(func(item *model.Item) bool {
		units = append(units, item.ToWeb())

		return true
	})

	return units
}

func getTypes(ctx *fiber.Ctx) error {
	return ctx.JSON(cot.Root)
}

func getLayers() []map[string]any {
	return []map[string]any{
		{
			"name":    "Google Hybrid",
			"url":     "http://mt{s}.google.com/vt/lyrs=y&x={x}&y={y}&z={z}&s=Galileo",
			"maxzoom": 20,
			"parts":   []string{"0", "1", "2", "3"},
		},
		{
			"name":    "OSM",
			"url":     "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
			"maxzoom": 19,
			"parts":   []string{"a", "b", "c"},
		},
		{
			"name":    "Opentopo.cz",
			"url":     "https://tile-{s}.opentopomap.cz/{z}/{x}/{y}.png",
			"maxzoom": 18,
			"parts":   []string{"a", "b", "c"},
		},
		{
			"name":    "Yandex maps",
			"url":     "https://core-renderer-tiles.maps.yandex.net/tiles?l=map&x={x}&y={y}&z={z}&scale=1&lang=ru_RU&projection=web_mercator",
			"maxzoom": 20,
		},
	}
}
