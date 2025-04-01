package main

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"

	"github.com/kdudkov/goatak/internal/repository"
)

const (
	UsernameKey = "username"
	SerialKey   = "sn"
)

func UserAuthHandler(r repository.UserRepository) fiber.Handler {
	return basicauth.New(basicauth.Config{
		Authorizer:      r.CheckAuth,
		ContextUsername: UsernameKey,
	})
}

func SSLCheckHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if c := ctx.Context(); c != nil {
			if tlsConnectionState := c.TLSConnectionState(); tlsConnectionState != nil {
				username, serial := getCertUser(tlsConnectionState)
				if app.users.IsValid(username, serial) {
					ctx.Locals(UsernameKey, username)
					ctx.Locals(SerialKey, serial)

					return ctx.Next()
				}

				app.logger.Warn(fmt.Sprintf("invalid user %s serial %s", username, serial))
			}
		}

		time.Sleep(3 * time.Second)

		return fiber.ErrUnauthorized
	}
}

func Username(c *fiber.Ctx) string {
	u := c.Locals(UsernameKey)

	if u == nil {
		return ""
	}

	return u.(string)
}
