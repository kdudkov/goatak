package log

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

func NewFiberLogger(api string, username func(c *fiber.Ctx) string) fiber.Handler {
	logger := slog.Default().With(slog.String("logger", api))

	return func(c *fiber.Ctx) error {
		start := time.Now()
		chainErr := c.Next()
		wt := time.Since(start)

		msg := fmt.Sprintf("%d %s %s %s", c.Response().StatusCode(), c.Method(), c.Path(), c.Request().URI().QueryArgs().String())
		l := logger

		if chainErr != nil {
			l = l.With(slog.Any("error", chainErr))
		}

		status := c.Response().StatusCode()

		var attrs []any
		if username != nil {
			attrs = []any{
				slog.String("client", c.IP()+":"+c.Port()),
				slog.Int("status", status),
				slog.String("user", username(c)),
				slog.Int64("ms", wt.Milliseconds()),
			}
		} else {
			attrs = []any{
				slog.String("client", c.IP()+":"+c.Port()),
				slog.Int("status", status),
				slog.Int64("ms", wt.Milliseconds()),
			}
		}

		switch {
		case status < 300:
			l.Debug(msg, attrs...)
		case status < 400:
			l.Info(msg, attrs...)
		default:
			l.Warn(msg, attrs...)
		}

		return nil
	}
}
