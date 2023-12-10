package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kdudkov/goatak/pkg/model"
)

const renewContacts = time.Second * 30

type RemoteAPI struct {
	host   string
	client *http.Client
	tls    bool
}

func NewRemoteAPI(host string) *RemoteAPI {
	return &RemoteAPI{
		host:   host,
		client: &http.Client{Timeout: time.Second * 5},
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

func (r *RemoteAPI) request(method, path string) (io.ReadCloser, error) {
	url := r.getURL(path)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Del("User-Agent")

	res, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status is %s", res.Status)
	}

	if res.Body == nil {
		return nil, fmt.Errorf("null body")
	}

	return res.Body, nil
}

func (r *RemoteAPI) getContacts() ([]*model.Contact, error) {
	b, err := r.request("GET", "/Marti/api/contacts/all")

	if b != nil {
		defer b.Close()
	}

	if err != nil {
		return nil, err
	}

	dat := make([]*model.Contact, 0)

	unm := json.NewDecoder(b)
	err = unm.Decode(&dat)

	return dat, err
}

func (app *App) periodicGetter(ctx context.Context) {
	ticker := time.NewTicker(renewContacts)
	defer ticker.Stop()

	d, _ := app.remoteAPI.getContacts()
	for _, c := range d {
		app.Logger.Debugf("contact %s %s", c.UID, c.Callsign)
		app.messages.Contacts.Store(c.UID, c)
	}

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dat, err := app.remoteAPI.getContacts()
			if err != nil {
				app.Logger.Warnf("error getting contacts: %s", err.Error())

				continue
			}

			for _, c := range dat {
				app.Logger.Debugf("contact %s %s", c.UID, c.Callsign)
				app.messages.Contacts.Store(c.UID, c)
			}
		}
	}
}
