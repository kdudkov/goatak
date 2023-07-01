package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/kdudkov/goatak/tlsutil"
	"io"
	"net/http"
	"time"
)

type Enroller struct {
	host   string
	port   int
	cl     *http.Client
	user   string
	passwd string
}

func NewEnroller(host, user, passwd string) *Enroller {
	tlsConf := &tls.Config{InsecureSkipVerify: true}

	return &Enroller{
		host:   host,
		user:   user,
		passwd: passwd,
		port:   8446,
		cl:     &http.Client{Timeout: time.Second * 30, Transport: &http.Transport{TLSClientConfig: tlsConf}},
	}
}

func (e *Enroller) baseUrl() string {
	return fmt.Sprintf("https://%s:%d", e.host, e.port)
}

func (e *Enroller) getConfig() (string, error) {
	req, err := http.NewRequest(http.MethodGet, e.baseUrl()+"/Marti/api/tls/config", http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Del("User-Agent")
	req.SetBasicAuth(e.user, e.passwd)
	res, err := e.cl.Do(req)

	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %d", res.StatusCode)
	}

	if res.Body == nil {
		return "", nil
	}

	defer res.Body.Close()

	d, err := io.ReadAll(res.Body)
	return string(d), err
}

func (e *Enroller) enrollCert(uid, version string) (*tls.Certificate, error) {
	conf, err := e.getConfig()
	if err != nil {
		return nil, err
	}
	fmt.Println(conf)

	csr, key := makeCsr(e.user)
	csr, _ = bytes.CutPrefix(csr, []byte("-----BEGIN CERTIFICATE REQUEST-----\n"))
	csr, _ = bytes.CutSuffix(csr, []byte("\n-----END CERTIFICATE REQUEST-----\n"))

	req, err := http.NewRequest(http.MethodPost, e.baseUrl()+"/Marti/api/tls/signClient/v2", bytes.NewReader(csr))
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("clientUID", uid)
	q.Add("version", version)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", "")
	req.SetBasicAuth(e.user, e.passwd)
	res, err := e.cl.Do(req)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", res.StatusCode)
	}

	var certs map[string]string

	if err := json.NewDecoder(res.Body).Decode(&certs); err != nil {
		return nil, err
	}

	if res.Body == nil {
		return nil, fmt.Errorf("empty response")
	}

	defer res.Body.Close()

	if c, ok := certs["signedCert"]; ok {
		cert, err := tlsutil.ParseCert(c)

		if err != nil {
			return nil, err
		}

		return &tls.Certificate{
			Certificate: [][]byte{cert.Raw},
			PrivateKey:  key,
			Leaf:        cert,
		}, nil
	}

	return nil, fmt.Errorf("no signed cert in responce")
}

func makeCsr(login string) ([]byte, *rsa.PrivateKey) {
	keyBytes, _ := rsa.GenerateKey(rand.Reader, 4096)

	subj := pkix.Name{
		CommonName: login,
	}

	template := x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, keyBytes)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes}), keyBytes
}
