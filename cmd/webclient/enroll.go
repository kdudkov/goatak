package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/kdudkov/goatak/pkg/tlsutil"
)

const minCertAge = time.Hour * 24

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

func (e *Enroller) getOrEnrollCert(uid, version string) (*tls.Certificate, error) {
	fname := fmt.Sprintf("%s_%s.p12", e.host, e.user)
	if cert, err := loadP12(fname, viper.GetString("ssl.password")); err == nil {
		e.logger.Infof("loading cert from file %s", fname)
		e.logger.Infof("cert is valid till %s", cert.Leaf.NotAfter)
		return cert, nil
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

	var cert *x509.Certificate
	ca := make([]*x509.Certificate, 0)

	defer res.Body.Close()

	for name, c := range certs {
		crt, err := tlsutil.ParseCert(c)

		if err != nil {
			return nil, err
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
		return nil, fmt.Errorf("no signed cert in answer")
	}

	if e.save {
		if err := e.saveP12(key, cert, ca); err != nil {
			e.logger.Errorf("%s", err)
		}
	}

	e.logger.Infof("cert enrollment successful")
	return &tls.Certificate{Certificate: [][]byte{cert.Raw}, PrivateKey: key, Leaf: cert}, nil
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

func (e *Enroller) saveP12(key interface{}, cert *x509.Certificate, ca []*x509.Certificate) error {
	f, err := os.Create(fmt.Sprintf("%s_%s.p12", e.host, e.user))
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := pkcs12.Encode(rand.Reader, key, cert, ca, viper.GetString("ssl.password"))
	_, _ = f.Write(data)
	return nil
}

func loadP12(filename, password string) (*tls.Certificate, error) {
	p12Data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	key, cert, _, err := pkcs12.DecodeChain(p12Data, password)
	if err != nil {
		return nil, err
	}

	if cert.NotAfter.Before(time.Now().Add(minCertAge)) {
		return nil, fmt.Errorf("cert is too old notAfter=(%s)", cert.NotAfter)
	}

	return &tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key.(crypto.PrivateKey),
		Leaf:        cert,
	}, nil
}
