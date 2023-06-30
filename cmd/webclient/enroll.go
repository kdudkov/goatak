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
	"io"
	"net/http"
	"time"
)

type Enroller struct {
	host    string
	port    int
	timeout time.Duration
	user    string
	passwd  string
}

func NewEnroller(host, user, passwd string) *Enroller {
	return &Enroller{
		host:    host,
		user:    user,
		passwd:  passwd,
		port:    8446,
		timeout: time.Second * 10,
	}
}

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

func (e *Enroller) baseUrl() string {
	return fmt.Sprintf("https://%s:%d", e.host, e.port)
}

func (e *Enroller) enrollCert() (*tls.Certificate, error) {
	cl := http.Client{Timeout: e.timeout}

	csr, key := makeCsr(e.user)
	csr, _ = bytes.CutPrefix(csr, []byte("-----BEGIN CERTIFICATE REQUEST-----\n"))
	csr, _ = bytes.CutSuffix(csr, []byte("\n-----END CERTIFICATE REQUEST-----"))

	req, err := http.NewRequest(http.MethodPost, e.baseUrl()+"/Marti/api/tls/signClient/v2", bytes.NewReader(csr))

	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(e.user, e.passwd)
	res, err := cl.Do(req)

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
		cert, err := parseCert(c)

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

func parseCert(s string) (*x509.Certificate, error) {
	bb := bytes.Buffer{}
	bb.WriteString("-----BEGIN CERTIFICATE-----\n")
	bb.WriteString(s)
	bb.WriteString("\n-----END CERTIFICATE-----")
	csrBlock, _ := pem.Decode(bb.Bytes())

	return x509.ParseCertificate(csrBlock.Bytes)
}
