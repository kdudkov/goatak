package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"github.com/kdudkov/goatak/pkg/tlsutil"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Enroller struct {
	logger *zap.SugaredLogger
	host   string
	port   int
	cl     *http.Client
	user   string
	passwd string
	save   bool
}

type CertificateConfig struct {
	XMLName      xml.Name `xml:"certificateConfig"`
	ValidityDays string   `xml:"validityDays,attr"`
	NameEntries  struct {
		NameEntry []struct {
			Value string `xml:"value,attr"`
			Name  string `xml:"name,attr"`
		} `xml:"nameEntry"`
	} `xml:"nameEntries"`
}

func NewEnroller(logger *zap.SugaredLogger, host, user, passwd string, save bool) *Enroller {
	tlsConf := &tls.Config{InsecureSkipVerify: true}
	return &Enroller{
		logger: logger,
		host:   host,
		user:   user,
		passwd: passwd,
		port:   8446,
		save:   save,
		cl:     &http.Client{Timeout: time.Second * 30, Transport: &http.Transport{TLSClientConfig: tlsConf}},
	}
}

func (e *Enroller) baseUrl() string {
	return fmt.Sprintf("https://%s:%d", e.host, e.port)
}

func (e *Enroller) getConfig() (*CertificateConfig, error) {
	e.logger.Infof("getting tls config")
	req, err := http.NewRequest(http.MethodGet, e.baseUrl()+"/Marti/api/tls/config", http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Del("User-Agent")
	req.SetBasicAuth(e.user, e.passwd)
	res, err := e.cl.Do(req)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", res.StatusCode)
	}

	if res.Body == nil {
		return nil, nil
	}

	defer res.Body.Close()

	dec := xml.NewDecoder(res.Body)
	conf := new(CertificateConfig)
	err = dec.Decode(conf)

	return conf, err
}

func (e *Enroller) enrollCert(uid, version string) (*tls.Certificate, error) {
	if cert, err := e.loadCert(); err == nil {
		if key, err := e.loadKey(); err == nil {
			e.logger.Infof("loading certs from file")
			return &tls.Certificate{Certificate: [][]byte{cert.Raw}, PrivateKey: key, Leaf: cert}, nil
		}
	}

	conf, err := e.getConfig()
	if err != nil {
		return nil, err
	}
	subj := new(pkix.Name)
	subj.CommonName = e.user

	if conf != nil {
		for _, c := range conf.NameEntries.NameEntry {
			switch c.Name {
			case "C":
				subj.Country = append(subj.Country, c.Value)
			case "O":
				subj.Organization = append(subj.Organization, c.Value)
			case "OU":
				subj.OrganizationalUnit = append(subj.OrganizationalUnit, c.Value)
			}
		}
	}

	csr, key := makeCsr(subj)
	e.logger.Infof("signing cert on server")
	req, err := http.NewRequest(http.MethodPost, e.baseUrl()+"/Marti/api/tls/signClient/v2", strings.NewReader(csr))
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("clientUID", uid)
	q.Add("version", version)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Del("User-Agent")
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

		if e.save {
			_ = e.saveCert(cert)
			_ = e.saveKey(key)
		}

		return &tls.Certificate{Certificate: [][]byte{cert.Raw}, PrivateKey: key, Leaf: cert}, nil
	}

	return nil, fmt.Errorf("no signed cert in responce")
}

func (e *Enroller) saveCert(cert *x509.Certificate) error {
	f, err := os.Create(fmt.Sprintf("%s_%s.pem", e.host, e.user))
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}

func (e *Enroller) saveKey(key *rsa.PrivateKey) error {
	f, err := os.Create(fmt.Sprintf("%s_%s.key", e.host, e.user))
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}

func (e *Enroller) loadCert() (*x509.Certificate, error) {
	f, err := os.Open(fmt.Sprintf("%s_%s.pem", e.host, e.user))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(b)
	return x509.ParseCertificate(block.Bytes)
}

func (e *Enroller) loadKey() (*rsa.PrivateKey, error) {
	f, err := os.Open(fmt.Sprintf("%s_%s.key", e.host, e.user))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(b)
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func makeCsr(subj *pkix.Name) (string, *rsa.PrivateKey) {
	keyBytes, _ := rsa.GenerateKey(rand.Reader, 4096)

	template := x509.CertificateRequest{
		Subject:            *subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, keyBytes)

	csr := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes}))

	csr = strings.ReplaceAll(csr, "-----BEGIN CERTIFICATE REQUEST-----\n", "")
	csr = strings.ReplaceAll(csr, "\n-----END CERTIFICATE REQUEST-----", "")

	return csr, keyBytes
}
