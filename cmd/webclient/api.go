package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/kdudkov/goatak/pkg/model"
	"io"
	"net/http"
	"time"
)

const renewContacts = time.Second * 30

type RemoteApi struct {
	host   string
	client *http.Client
	tls    bool
}

func NewRemoteApi(host string) *RemoteApi {
	return &RemoteApi{
		host:   host,
		client: &http.Client{Timeout: time.Second * 5},
	}
}

func (r *RemoteApi) SetTls(config *tls.Config) {
	r.client.Transport = &http.Transport{TLSClientConfig: config}
	r.tls = true
}

func (r *RemoteApi) getUrl(path string) string {
	if r.tls {
		return fmt.Sprintf("https://%s:8443%s", r.host, path)
	}
	return fmt.Sprintf("http://%s:8080%s", r.host, path)
}

func (r *RemoteApi) request(method, path string) (io.ReadCloser, error) {
	url := r.getUrl(path)
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

func (r *RemoteApi) getContacts() ([]*model.Contact, error) {
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

func (r *RemoteApi) getTest() error {
	b, err := r.request("GET", "/Marti/api/resources/7a538172f4cc8541504db5598a723e28510ad0f93d8694d5c3cc53c6d0501f67")

	if b != nil {
		defer b.Close()
	}

	if err != nil {
		return err
	}

	dat, err := io.ReadAll(b)
	fmt.Println(string(dat))
	return err
}

func (app *App) periodicGetter(ctx context.Context) {
	ticker := time.NewTicker(renewContacts)
	defer ticker.Stop()

	d, _ := app.remoteApi.getContacts()
	for _, c := range d {
		app.Logger.Debugf("contact %s %s", c.Uid, c.Callsign)
		app.messages.Contacts.Store(c.Uid, c)
	}

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dat, err := app.remoteApi.getContacts()
			if err != nil {
				app.Logger.Warnf("error getting contacts: %s", err.Error())
				continue
			}

			for _, c := range dat {
				app.Logger.Debugf("contact %s %s", c.Uid, c.Callsign)
				app.messages.Contacts.Store(c.Uid, c)
			}
		}
	}
}
