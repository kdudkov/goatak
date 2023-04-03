package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func getBody(res *http.Response) ([]byte, error) {
	if res == nil {
		return nil, fmt.Errorf("empty response")
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", res.StatusCode)
	}

	if res.Body == nil {
		return nil, fmt.Errorf("empty response")
	}

	defer res.Body.Close()

	return io.ReadAll(res.Body)
}

func (app *App) enrollCert(user, password string) error {
	cl := http.Client{Timeout: time.Second}
	baseUrl := fmt.Sprintf("https://%s:8446", app.host)

	res, err := cl.Get(baseUrl + "/Marti/api/tls/config")

	if err != nil {
		return err
	}

	dat, err := getBody(res)

	if err != nil {
		return err
	}

	fmt.Println(string(dat))
	return nil
}
