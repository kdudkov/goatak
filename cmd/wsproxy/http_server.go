package main

import (
	"github.com/kdudkov/goatak/cmd/wsproxy/tak_ws"
	"github.com/kdudkov/goatak/pkg/log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func NewHttp(app *App) *fiber.App {
	srv := fiber.New(fiber.Config{EnablePrintRoutes: false, DisableStartupMessage: true})
	srv.Use(log.NewFiberLogger(nil))
	srv.Get("/", getIndexHandler(app))
	srv.Get("/takproto/1", getTakWsHandler(app))
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

func getTakWsHandler(app *App) fiber.Handler {
	return websocket.New(func(ws *websocket.Conn) {
		defer ws.Close()

		app.logger.Info("WS connection from " + ws.RemoteAddr().String())
		name := "ws:" + ws.RemoteAddr().String()
		w := tak_ws.New(name, nil, ws, app.ProcessCotFromWSClient)

		app.AddClientHandler(w)
		w.Listen()
		app.logger.Info("ws disconnected")
		app.RemoveClientHandler(w.GetName())
	})
}
