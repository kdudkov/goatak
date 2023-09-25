package main

import (
	"errors"
	"github.com/aofei/air"
	"net/http"
)

func SslCheckHandler(app *App) air.Gas {
	return func(next air.Handler) air.Handler {
		return func(req *air.Request, res *air.Response) error {
			if h := req.HTTPRequest(); h != nil {
				if h.TLS != nil {
					user, serial := getCertUser(h.TLS)
					if app.users.UserIsValid(user, serial) {
						req.SetValue("user", user)
						return next(req, res)
					} else {
						app.Logger.Warnf("invalid user %s serial %s", user, serial)
					}
				}
			}
			res.Status = http.StatusUnauthorized
			return errors.New("error")
		}
	}
}
