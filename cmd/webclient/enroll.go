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
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kdudkov/goatak/pkg/tlsutil"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"software.sslmate.com/src/go-pkcs12"
)

const minCertAge = time.Hour * 24

type Enroller struct {
	logger *zap.SugaredLogger
	host   string
	port   int
	client *http.Client
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
		client: &http.Client{Timeout: time.Second * 30, Transport: &http.Transport{TLSClientConfig: tlsConf}},
	}
}

func (e *Enroller) getUrl(path string) string {
	return fmt.Sprintf("https://%s:%d%s", e.host, e.port, path)
}

func (e *Enroller) request(method, path string, args map[string]string, body io.Reader) (io.ReadCloser, error) {
	url := e.getUrl(path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(e.user, e.passwd)
	req.Header.Del("User-Agent")

	if args != nil && len(args) > 0 {
		q := req.URL.Query()

		for k, v := range args {
			q.Add(k, v)

		}
		req.URL.RawQuery = q.Encode()
	}

	res, err := e.client.Do(req)
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

func (e *Enroller) getConfig() (*CertificateConfig, error) {
	e.logger.Infof("getting tls config")
	body, err := e.request(http.MethodGet, "/Marti/api/tls/config", nil, nil)

	if err != nil {
		return nil, err
	}

	defer body.Close()

	dec := xml.NewDecoder(body)
	conf := new(CertificateConfig)
	err = dec.Decode(conf)

	return conf, err
}

func (e *Enroller) getOrEnrollCert(uid, version string) (*tls.Certificate, []*x509.Certificate, error) {
	fname := fmt.Sprintf("%s_%s.p12", e.host, e.user)
	if cert, cas, err := loadP12(fname, viper.GetString("ssl.password")); err == nil {
		e.logger.Infof("loading cert from file %s", fname)

		return cert, cas, nil
	}

	conf, err := e.getConfig()
	if err != nil {
		return nil, nil, err
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
	args := map[string]string{"clientUID": uid, "version": version}
	body, err := e.request(http.MethodPost, "/Marti/api/tls/signClient/v2", args, strings.NewReader(csr))

	if err != nil {
		return nil, nil, err
	}

	defer body.Close()

	var certs map[string]string

	if err := json.NewDecoder(body).Decode(&certs); err != nil {
		return nil, nil, err
	}

	var cert *x509.Certificate

	ca := make([]*x509.Certificate, 0)

	for name, c := range certs {
		crt, err := tlsutil.ParseCert(c)
		if err != nil {
			return nil, nil, err
		}

		if name == "signedCert" {
			cert = crt

			continue
		}

		if strings.HasPrefix(name, "ca") {
			ca = append(ca, crt)
		}
	}

	if cert == nil {
		return nil, nil, fmt.Errorf("no signed cert in answer")
	}

	tlsutil.LogCert(e.logger, "signed cert", cert)

	if e.save {
		if err := e.saveP12(key, cert, ca); err != nil {
			e.logger.Errorf("%s", err)
		}
	}

	e.logger.Infof("cert enrollment successful")

	if err := e.getProfile(uid); err != nil {
		e.logger.Warnf("%s", err.Error())
	}

	return &tls.Certificate{Certificate: [][]byte{cert.Raw}, PrivateKey: key, Leaf: cert}, ca, nil
}

func (e *Enroller) getProfile(uid string) error {
	args := map[string]string{"clientUID": uid}
	body, err := e.request(http.MethodGet, "/Marti/api/tls/profile/enrollment", args, nil)

	if err != nil {
		return err
	}

	defer body.Close()

	f, err := os.Create(e.host + ".zip")
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.Copy(f, body)
	return err
}

func makeCsr(subj *pkix.Name) (string, *rsa.PrivateKey) {
	keyBytes, _ := rsa.GenerateKey(rand.Reader, 4096)

	template := x509.CertificateRequest{
		Subject:            *subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, keyBytes)

	csr := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes, Headers: nil}))

	csr = strings.ReplaceAll(csr, "-----BEGIN CERTIFICATE REQUEST-----\n", "")
	csr = strings.ReplaceAll(csr, "\n-----END CERTIFICATE REQUEST-----", "")

	return csr, keyBytes
}

func (e *Enroller) saveP12(key interface{}, cert *x509.Certificate, ca []*x509.Certificate) error {
	f, err := os.Create(fmt.Sprintf("%s_%s.p12", e.host, e.user))
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := pkcs12.Modern.Encode(key, cert, ca, viper.GetString("ssl.password"))
	if err != nil {
		return err
	}
	_, _ = f.Write(data)
	return nil
}

func loadP12(filename, password string) (*tls.Certificate, []*x509.Certificate, error) {
	p12Data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	key, cert, cas, err := pkcs12.DecodeChain(p12Data, password)
	if err != nil {
		return nil, nil, err
	}

	if cert.NotAfter.Before(time.Now().Add(minCertAge)) {
		return nil, nil, fmt.Errorf("cert is too old notAfter=(%s)", cert.NotAfter)
	}

	return &tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key,
		Leaf:        cert,
	}, cas, nil
}
