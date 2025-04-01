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

	"github.com/kdudkov/goutils/request"

	"github.com/kdudkov/goatak/pkg/model"
	mp "github.com/kdudkov/goatak/pkg/model"
)

const (
	renewContacts = time.Second * 30
	httpTimeout   = time.Second * 3
)

type RemoteAPI struct {
	logger *slog.Logger
	host   string
	client *http.Client
	tls    bool
}

type JSONResult[T any] struct {
	Count int `json:"resultCount"`
	Data  []T `json:"results"`
}

type PackageInfo struct {
	UID                string    `json:"UID"`
	SubmissionDateTime time.Time `json:"SubmissionDateTime"`
	Keywords           []string  `json:"Keywords"`
	MIMEType           string    `json:"MIMEType"`
	Size               int       `json:"Size"`
	SubmissionUser     string    `json:"SubmissionUser"`
	PrimaryKey         int       `json:"PrimaryKey"`
	Hash               string    `json:"Hash"`
	CreatorUID         string    `json:"CreatorUid"`
	Scope              string    `json:"Scope"`
	Name               string    `json:"FileName"`
	Tool               string    `json:"Tool"`
	Expiration         int64     `json:"Expiration"`
}

func NewRemoteAPI(host string) *RemoteAPI {
	return &RemoteAPI{
		host:   host,
		logger: slog.Default().With("logger", "remote_api"),
		client: &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: httpTimeout}},
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

func (r *RemoteAPI) GetMissions(ctx context.Context) ([]*mp.MissionDTO, error) {
	res := new(mp.Answer[[]*mp.MissionDTO])

	err := r.request("/Marti/api/missions").GetJSON(ctx, &res)

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

	err := r.request(fmt.Sprintf("/Marti/api/missions/%s/subscriptions", name)).GetJSON(ctx, &res)

	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.New("nil")
	}

	return res.Data, err
}

func (r *RemoteAPI) GetSubscriptionRoles(ctx context.Context, name string) (string, error) {
	b, err := r.request("/Marti/api/groups/all").Do(ctx)

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
	b, err := r.request("/Marti/api/missions/" + name).
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
	b, err := r.request("/Marti/api/missions/" + name + "/subscription").
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

func (r *RemoteAPI) Search(ctx context.Context) ([]*PackageInfo, error) {
	res := new(JSONResult[*PackageInfo])
	err := r.request("/Marti/sync/search").GetJSON(ctx, &res)

	if err != nil {
		return nil, err
	}

	return res.Data, nil
}

func (r *RemoteAPI) GetFile(ctx context.Context, hash string, f func(r io.Reader) error) error {
	res, err := r.request("/Marti/sync/content").Args(map[string]string{"hash": hash}).DoRes(ctx)

	if err != nil {
		return err
	}

	for k, v := range res.Header {
		r.logger.Debug(fmt.Sprintf("%s: %s\n", k, v))
	}

	if res.Body != nil {
		defer res.Body.Close()

		if f != nil {
			return f(res.Body)
		}
	}

	return nil
}
