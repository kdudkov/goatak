package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/kdudkov/goatak/internal/config"
	"github.com/kdudkov/goatak/pkg/model"
)

type TestApp struct {
	*App
	api *AdminAPI
}

func Device(login, pass string, admin, disabled bool) *model.Device {
	d := new(model.Device)
	d.Login = login
	if err := d.SetPassword(pass); err != nil {
		panic(err)
	}
	d.Disabled = disabled
	d.Admin = admin

	return d
}

func NewTestApp() *TestApp {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	cfg := config.NewAppConfig()
	cfg.Set("db", ":memory:")
	cfg.Set("delay", false)

	app := &TestApp{
		App: NewApp(cfg),
	}

	if err := app.items.Start(); err != nil {
		panic(err)
	}

	if err := app.users.Start(); err != nil {
		panic(err)
	}

	app.dbm.Save(Device("adm1", "111", true, false))
	app.dbm.Save(Device("adm2", "222", true, true))
	app.dbm.Save(Device("usr1", "1", false, false))
	app.dbm.Save(Device("usr2", "2", false, true))

	srv := &HttpServer{
		log:         app.logger.With("logger", "http"),
		listeners:   make(map[string]Listener),
		userManager: app.users,
		tokenKey:    []byte("111"),
		tokenMaxAge: time.Hour,
		loginUrl:    "/login",
		noAuth:      nil,
	}

	app.api = srv.NewAdminAPI(app.App, "localhost:1234", "")

	return app
}

func (app *TestApp) Req(method, url, token string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	return app.api.f.Test(req, 3000)
}

func (app *TestApp) PostJSON(url, token string, obj any) (*http.Response, error) {
	d, err := json.Marshal(obj)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(d))

	if err != nil {
		return nil, err
	}

	req.Header.Add(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	req.Header.Add(fiber.HeaderAccept, fiber.MIMEApplicationJSON)

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	return app.api.f.Test(req, 3000)
}

func TestLogin(t *testing.T) {
	app := NewTestApp()

	for _, d := range []struct {
		login string
		psw   string
		ok    bool
	}{
		{"adm1", "111", true},
		{"adm1", "1111", false},
		{"adm2", "222", false},
		{"usr1", "1", false},
		{"usr2", "2", false},
	} {
		t.Run("login_as_"+d.login, func(t *testing.T) {
			resp, err := app.PostJSON("/token", "", fiber.Map{"login": d.login, "password": d.psw})
			require.NoError(t, err)

			if d.ok {
				require.Equal(t, fiber.StatusOK, resp.StatusCode)
			} else {
				require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			}
		})
	}
}

func TestLoginGetToken(t *testing.T) {
	app := NewTestApp()

	resp, err := app.Req("GET", "/", "", nil)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusFound, resp.StatusCode)

	resp, err = app.PostJSON("/token", "", fiber.Map{"login": "adm1", "password": "111"})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	m := make(map[string]string)
	require.NotNil(t, resp.Body)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&m))

	token := m["token"]

	require.NotEmpty(t, token)

	resp, err = app.Req("GET", "/", token, nil)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	resp, err = app.PostJSON("/token", "", fiber.Map{"login": "admin", "password": "1234"})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}
