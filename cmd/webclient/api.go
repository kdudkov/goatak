package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kdudkov/goatak/pkg/model"
	"net/http"
	"time"
)

const renewContacts = time.Second * 30

func (app *App) getContacts() error {
	url := ""
	if app.tls {
		url = fmt.Sprintf("https://%s:8443/Marti/api/contacts/all", app.host)
	} else {
		url = fmt.Sprintf("http://%s:8080/Marti/api/contacts/all", app.host)
	}

	res, err := app.client.Get(url)

	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status is %s", res.Status)
	}

	if res.Body == nil {
		return fmt.Errorf("null body")
	}

	defer res.Body.Close()

	dat := make([]*model.Contact, 0)

	unm := json.NewDecoder(res.Body)

	if err := unm.Decode(&dat); err != nil {
		return err
	}

	app.Logger.Debugf("got %d contacts", len(dat))
	for _, c := range dat {
		app.Logger.Debugf("contact %s %s", c.Uid, c.Callsign)
		app.messages.Contacts.Store(c.Uid, c)
	}

	return nil
}

func (app *App) periodicGetter(ctx context.Context) {
	ticker := time.NewTicker(renewContacts)
	defer ticker.Stop()

	_ = app.getContacts()

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := app.getContacts(); err != nil {
				app.Logger.Warnf("contacts get error: %s", err.Error())
			}
		}
	}
}
