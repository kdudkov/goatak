package main

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/golang-jwt/jwt/v5"

	"github.com/kdudkov/goatak/pkg/model"
)

const (
	header      = "Authorization"
	cookieName  = "token"
	bearer      = "Bearer"
	UsernameKey = "username"
	UserKey     = "user"
	SerialKey   = "sn"
)

var (
	noTokenErr = errors.New("api token required")
	badToken   = errors.New("bad token")
	badUser    = errors.New("invalid user")
)

func (h *HttpServer) HeaderAuth(c *fiber.Ctx) error {
	user, err := h.checkToken(getToken(c))

	if err != nil {
		c.ClearCookie(cookieName)
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	c.Locals(UsernameKey, user.Login)
	c.Locals(UserKey, user)

	return c.Next()

}

func (h *HttpServer) CookieAuth(c *fiber.Ctx) error {
	if c.Path() == h.loginUrl {
		return c.Next()
	}

	for _, p := range h.noAuth {
		if strings.HasPrefix(c.Path(), p) {
			return c.Next()
		}
	}

	user, err := h.checkToken(getToken(c))

	if err != nil {
		c.ClearCookie(cookieName)
		return c.Redirect(h.loginUrl)
	}

	c.Locals(UsernameKey, user.Login)
	c.Locals(UserKey, user)

	return c.Next()
}

func (h *HttpServer) checkToken(tokenStr string) (*model.Device, error) {
	if tokenStr == "" {
		return nil, noTokenErr
	}

	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return h.tokenKey, nil
	})

	if err != nil {
		h.log.With(slog.String("logger", "auth")).Warn("token parse error", slog.Any("error", err))

		if errors.Is(err, jwt.ErrSignatureInvalid) || errors.Is(err, jwt.ErrTokenExpired) {
			return nil, badToken
		}

		return nil, err
	}

	if !token.Valid {
		return nil, badToken
	}

	if u := h.userManager.Get(claims.Subject); u != nil && !u.Disabled && u.CanLogIn() {
		return u, nil
	}

	return nil, badUser
}

func (h *HttpServer) DeviceAuthHandler() fiber.Handler {
	return basicauth.New(basicauth.Config{
		Authorizer:      h.userManager.CheckAuth,
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

func User(c *websocket.Conn) *model.Device {
	val := c.Locals(UserKey)

	if u, ok := val.(*model.Device); ok {
		return u
	}

	return nil
}

func getToken(c *fiber.Ctx) string {
	if s := c.Get(header); s != "" {
		l := len(bearer)

		if len(s) > l+1 && s[:l] == bearer {
			return s[l+1:]
		}
	}

	return c.Cookies(cookieName)
}

func generateToken(username string, key []byte, t time.Duration) (string, error) {
	claims := &jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(t)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
