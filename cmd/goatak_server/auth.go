package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/kdudkov/goatak/internal/repository"
)

const (
	UsernameKey = "username"
)

func getUserAuth(r repository.UserRepository) fiber.Handler {
	return basicauth.New(basicauth.Config{
		Authorizer:      r.CheckUserAuth,
		ContextUsername: UsernameKey,
	})
}

func Username(c *fiber.Ctx) string {
	u := c.Locals(UsernameKey)

	if u == nil {
		return ""
	}

	return u.(string)
}
