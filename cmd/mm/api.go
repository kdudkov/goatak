package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kdudkov/goatak/internal/client"
	mp "github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/model"
)

const renewContacts = time.Second * 30

type RemoteAPI struct {
	logger *slog.Logger
	host   string
	client *http.Client
	tls    bool
}

func NewRemoteAPI(host string) *RemoteAPI {
	return &RemoteAPI{
		host:   host,
		logger: slog.Default().With("logger", "remote_api"),
		client: &http.Client{Timeout: time.Second * 3},
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

func (r *RemoteAPI) GetMissions(ctx context.Context) ([]*mp.MissionDTO, error) {
	res := new(mp.Answer[[]*mp.MissionDTO])

	err := client.NewRequest(r.client, r.getURL("/Marti/api/missions")).
		GetJSON(ctx, &res)

	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.New("nil")
	}

	return res.Data, err
}

func (r *RemoteAPI) GetSubscriptions(ctx context.Context, name string) ([]string, error) {
	res := new(mp.Answer[[]string])

	err := client.NewRequest(r.client, r.getURL(fmt.Sprintf("/Marti/api/missions/%s/subscriptions", name))).
		GetJSON(ctx, &res)

	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.New("nil")
	}

	return res.Data, err
}

func (r *RemoteAPI) GetSubscriptionRoles(ctx context.Context, name string) (string, error) {
	b, err := client.NewRequest(r.client, r.getURL("/Marti/api/groups/all")).
		Do(ctx)

	if err != nil {
		return "", err
	}

	if b == nil {
		return "", nil
	}

	defer b.Close()

	d, err := io.ReadAll(b)

	return string(d), err
}

func (r *RemoteAPI) CreateMission(ctx context.Context, name string, uid string) error {
	b, err := client.NewRequest(r.client, r.getURL("/Marti/api/missions/"+name)).
		Put().
		Args(map[string]string{"creatorUid": uid, "tool": "public", "group": "__ANON__"}).
		Do(ctx)

	if err != nil {
		return err
	}

	if b == nil {
		return nil
	}

	defer b.Close()

	d, err := io.ReadAll(b)

	fmt.Println(string(d))

	return nil
}

func (r *RemoteAPI) Subscribe(ctx context.Context, name string, uid string) error {
	b, err := client.NewRequest(r.client, r.getURL("/Marti/api/missions/"+name+"/subscription")).
		Put().
		Args(map[string]string{"uid": uid}).
		Do(ctx)

	if err != nil {
		return err
	}

	if b == nil {
		return nil
	}

	defer b.Close()

	d, err := io.ReadAll(b)

	fmt.Println(string(d))

	return nil
}
