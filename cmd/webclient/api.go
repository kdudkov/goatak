package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kdudkov/goutils/request"

	"github.com/kdudkov/goatak/pkg/model"
)

const (
	renewContacts = time.Second * 120
	httpTimeout   = time.Second * 5
)

type RemoteAPI struct {
	logger *slog.Logger
	host   string
	client *http.Client
	tls    bool
}

func NewRemoteAPI(host string, logger *slog.Logger) *RemoteAPI {
	return &RemoteAPI{
		logger: logger,
		host:   host,
		client: &http.Client{Timeout: httpTimeout},
	}
}

func (r *RemoteAPI) SetTLS(config *tls.Config) {
	r.client.Transport = &http.Transport{TLSClientConfig: config}
	r.tls = true
}

func (r *RemoteAPI) getURL(path string) string {
	if r.tls {
		return fmt.Sprintf("https://%s:8443%s", r.host, path)
	}

	return fmt.Sprintf("http://%s:8080%s", r.host, path)
}

func (r *RemoteAPI) request(url string) *request.Request {
	return request.New(r.client, r.logger).URL(r.getURL(url))
}

func (r *RemoteAPI) getContacts(ctx context.Context) ([]*model.Contact, error) {
	dat := make([]*model.Contact, 0)

	err := r.request("/Marti/api/contacts/all").GetJSON(ctx, &dat)

	return dat, err
}

func (r *RemoteAPI) getConfig(ctx context.Context, uid string) (string, error) {

	resp, err := r.request("/Marti/api/device/profile/connection").Args(map[string]string{"clientUid": uid}).DoRes(ctx)

	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", errors.New("bad status " + resp.Status)
	}

	if resp.Body == nil {
		return "", errors.New("null body")
	}

	defer resp.Body.Close()

	disp := resp.Header.Get(fiber.HeaderContentDisposition)

	fname := fmt.Sprintf("config_%s.zip", uid)

	if disp != "" {
		r.logger.Info("Content: " + disp)
	}

	fd, err := os.Create(fname)

	if err != nil {
		return "", err
	}

	defer fd.Close()

	_, err = io.Copy(fd, resp.Body)

	return fname, err
}

func (app *App) periodicGetter(ctx context.Context) {
	ticker := time.NewTicker(renewContacts)
	defer ticker.Stop()

	d, _ := app.remoteAPI.getContacts(ctx)
	for _, c := range d {
		app.logger.Debug(fmt.Sprintf("contact %s %s", c.UID, c.Callsign))
		app.chatMessages.Contacts.Store(c.UID, c)
	}

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dat, err := app.remoteAPI.getContacts(ctx)
			if err != nil {
				app.logger.Warn("error getting contacts", slog.Any("error", err))

				continue
			}

			for _, c := range dat {
				app.logger.Debug(fmt.Sprintf("contact %s %s", c.UID, c.Callsign))
				app.chatMessages.Contacts.Store(c.UID, c)
			}
		}
	}
}
