package log

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

type LoggerConfig struct {
	Name       string
	UserGetter func(c *fiber.Ctx) string
	Level      slog.Level
}

func NewFiberLogger(conf *LoggerConfig) fiber.Handler {
	if conf == nil {
		conf = &LoggerConfig{Name: "http", Level: slog.LevelInfo}
	}

	logger := slog.Default().With(slog.String("logger", conf.Name))

	return func(c *fiber.Ctx) error {
		start := time.Now()
		chainErr := c.Next()
		wt := time.Since(start)
		status := c.Response().StatusCode()

		l := logger

		if chainErr != nil {
			var e *fiber.Error
			if errors.As(chainErr, &e) {
				status = e.Code
			}

			l = l.With(slog.Any("error", chainErr))
		}

		msg := fmt.Sprintf("%d %s %s %s", status, c.Method(), c.Path(), c.Request().URI().QueryArgs().String())

		var attrs []any
		if conf.UserGetter != nil {
			attrs = []any{
				slog.String("client", c.IP()+":"+c.Port()),
				slog.Int("status", status),
				slog.String("user", conf.UserGetter(c)),
				slog.Int64("ms", wt.Milliseconds()),
			}
		} else {
			attrs = []any{
				slog.String("client", c.IP()+":"+c.Port()),
				slog.Int("status", status),
				slog.Int64("ms", wt.Milliseconds()),
			}
		}

		var lvl slog.Level

		switch {
		case status < 300:
			lvl = conf.Level
		case status < 400:
			lvl = slog.LevelWarn
		default:
			lvl = slog.LevelError
		}

		l.Log(context.Background(), lvl, msg, attrs...)

		return chainErr
	}
}
