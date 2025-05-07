package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/pprof"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/template/html/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kdudkov/goatak/pkg/cot"
)

type LocalAPI struct {
	f    *fiber.App
	addr string
}

func (api *LocalAPI) Address() string {
	return api.addr
}

func (api *LocalAPI) Listen() error {
	return api.f.Listen(api.addr)
}

func (h *HttpServer) NewLocalAPI(app *App, addr string) *LocalAPI {
	api := &LocalAPI{addr: addr}
	h.listeners["local api calls"] = api

	engine := html.NewFileSystem(http.FS(templates), ".html")

	engine.Delims("[[", "]]")

	api.f = fiber.New(fiber.Config{EnablePrintRoutes: false, DisableStartupMessage: true, Views: engine})

	api.f.Post("/cot", getCotPostHandler(app))
	api.f.Get("/stack", getStackHandler())
	api.f.Get("/metrics", getMetricsHandler())

	return api
}

func getCotPostHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		c := new(cot.CotMessage)

		if err := json.Unmarshal(ctx.Body(), c); err != nil {
			app.logger.Error("cot decode error", slog.Any("error", err))

			return err
		}

		app.NewCotMessage(c)

		return nil
	}
}

func getStackHandler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return pprof.Lookup("goroutine").WriteTo(ctx.Response().BodyWriter(), 1)
	}
}

func getMetricsHandler() fiber.Handler {
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})

	return adaptor.HTTPHandler(handler)
}
