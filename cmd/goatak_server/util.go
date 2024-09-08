package main

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func logParams(log *slog.Logger, ctx *fiber.Ctx) {
	var params []string

	for k, v := range ctx.AllParams() {
		params = append(params, k+"="+v)
	}

	log.Info("params: " + strings.Join(params, ","))
}

func queryIgnoreCase(c *fiber.Ctx, name string) string {
	nn := strings.ToLower(name)
	for k, v := range c.Queries() {
		if strings.ToLower(k) == nn {
			return v
		}
	}

	return ""
}
