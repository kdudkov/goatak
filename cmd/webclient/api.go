package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kdudkov/goatak/internal/client"
	mp "github.com/kdudkov/goatak/internal/model"
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

func (r *RemoteAPI) getContacts(ctx context.Context) ([]*model.Contact, error) {
	dat := make([]*model.Contact, 0)

	err := client.NewRequest(r.client, r.getURL("/Marti/api/contacts/all")).GetJSON(ctx, &dat)

	return dat, err
}

func (r *RemoteAPI) NewMission(ctx context.Context, uid string) error {
	b, err := client.NewRequest(r.client, r.getURL("/Marti/api/missions/test_mission_55567/subscription")).
		Put().
		Args(map[string]string{"uid": uid}).
		Do(ctx)

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

func (r *RemoteAPI) GetMissions(ctx context.Context) error {
	res := make(map[string]any)

	err := client.NewRequest(r.client, r.getURL("/Marti/api/missions")).
		GetJSON(ctx, &res)

	if d, ok := res["data"]; ok {
		if n, ok1 := d.([]*mp.MissionDTO); ok1 {
			for _, nn := range n {
				fmt.Println(nn.Name)
			}
		}
	}

	return err
}

func (app *App) periodicGetter(ctx context.Context) {
	ticker := time.NewTicker(renewContacts)
	defer ticker.Stop()

	d, _ := app.remoteAPI.getContacts(ctx)
	for _, c := range d {
		app.Logger.Debugf("contact %s %s", c.UID, c.Callsign)
		app.messages.Contacts.Store(c.UID, c)
	}

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dat, err := app.remoteAPI.getContacts(ctx)
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
