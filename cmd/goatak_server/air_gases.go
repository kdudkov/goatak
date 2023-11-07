package main

import (
	"errors"
	"github.com/aofei/air"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func SslCheckHandlerGas(app *App) air.Gas {
	return func(next air.Handler) air.Handler {
		return func(req *air.Request, res *air.Response) error {
			if h := req.HTTPRequest(); h != nil {
				if h.TLS != nil {
					user, serial := getCertUser(h.TLS)
					if app.users.UserIsValid(user, serial) {
						req.SetValue(usernameKey, user)
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

func LoggerGas(log *zap.SugaredLogger) air.Gas {
	return func(next air.Handler) air.Handler {
		return func(req *air.Request, res *air.Response) (err error) {
			startTime := time.Now()
			res.Defer(func() {
				endTime := time.Now()
				user := getUsernameFromReq(req)

				log.With(zap.String("user", user), zap.Int("status", res.Status)).Infof("%s %s, client: %s, time :%s",
					req.Method, req.Path, req.ClientAddress(), endTime.Sub(startTime))
			})

			return next(req, res)
		}
	}
}
