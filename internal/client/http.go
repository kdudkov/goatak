package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type Request struct {
	client *http.Client
	url    string
	method string
	login  string
	passw  string
	body   io.Reader
	args   map[string]string
	logger *slog.Logger
}

func NewRequest(c *http.Client, logger *slog.Logger) *Request {
	return &Request{client: c, method: "GET", logger: logger}
}

func (r *Request) URL(url string) *Request {
	r.url = url

	return r
}

func (r *Request) Put() *Request {
	r.method = "PUT"

	return r
}

func (r *Request) Post() *Request {
	r.method = "POST"

	return r
}

func (r *Request) Auth(login, passw string) *Request {
	r.login = login
	r.passw = passw

	return r
}

func (r *Request) Args(args map[string]string) *Request {
	r.args = args

	return r
}

func (r *Request) Body(body io.Reader) *Request {
	r.body = body

	return r
}

func (r *Request) DoRes(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, r.method, r.url, r.body)
	if err != nil {
		return nil, err
	}

	if r.login != "" {
		req.SetBasicAuth(r.login, r.passw)
	}

	req.Header.Del("User-Agent")

	if len(r.args) > 0 {
		q := req.URL.Query()

		for k, v := range r.args {
			q.Add(k, v)
		}

		req.URL.RawQuery = q.Encode()
	}

	res, err := r.client.Do(req)
	if err != nil {
		if r.logger != nil {
			r.logger.Info(fmt.Sprintf("%s %s - error %s", r.method, req.URL, err.Error()))
		}

		return res, err
	}

	if r.logger != nil {
		r.logger.Info(fmt.Sprintf("%s %s - %d", r.method, req.URL, res.StatusCode))
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return res, fmt.Errorf("status is %s", res.Status)
	}

	return res, nil
}

func (r *Request) Do(ctx context.Context) (io.ReadCloser, error) {
	res, err := r.DoRes(ctx)

	if err != nil {
		return nil, err
	}

	if res.Body == nil {
		return nil, fmt.Errorf("null body")
	}

	return res.Body, nil
}

func (r *Request) GetJSON(ctx context.Context, obj any) error {
	b, err := r.Do(ctx)

	if err != nil {
		return err
	}

	dec := json.NewDecoder(b)

	return dec.Decode(obj)
}
