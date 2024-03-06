package client

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/kdudkov/goatak/pkg/tlsutil"
)

const minCertAge = time.Hour * 24
const keySize = 4096

type Enroller struct {
	logger *slog.Logger
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

func NewEnroller(host, user, passwd string, save bool) *Enroller {
	tlsConf := &tls.Config{InsecureSkipVerify: true}

	return &Enroller{
		logger: slog.Default().With("logger", "enroller"),
		host:   host,
		user:   user,
		passwd: passwd,
		port:   8446,
		save:   save,
		client: &http.Client{Timeout: time.Second * 30, Transport: &http.Transport{TLSClientConfig: tlsConf}},
	}
}

func (e *Enroller) getURL(path string) string {
	return fmt.Sprintf("https://%s:%d%s", e.host, e.port, path)
}

func (e *Enroller) getConfig(ctx context.Context) (*CertificateConfig, error) {
	e.logger.Info("getting tls config")

	body, err := NewRequest(e.client, e.getURL("/Marti/api/tls/config")).
		Auth(e.user, e.passwd).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	defer body.Close()

	dec := xml.NewDecoder(body)
	conf := new(CertificateConfig)
	err = dec.Decode(conf)

	return conf, err
}

func (e *Enroller) GetOrEnrollCert(ctx context.Context, uid, version string) (*tls.Certificate, []*x509.Certificate, error) {
	fname := fmt.Sprintf("%s_%s.p12", e.host, e.user)
	if cert, cas, err := LoadP12(fname, viper.GetString("ssl.password")); err == nil {
		e.logger.Info("loading cert from file " + fname)

		return cert, cas, nil
	}

	conf, err := e.getConfig(ctx)
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

	e.logger.Info("signing cert on server")

	args := map[string]string{"clientUID": uid, "version": version}

	var certs map[string]string

	err = NewRequest(e.client, e.getURL("/Marti/api/tls/signClient/v2")).Auth(e.user, e.passwd).
		Post().
		Args(args).
		Body(strings.NewReader(csr)).
		GetJSON(ctx, &certs)

	if err != nil {
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
			e.logger.Error("error", "error", err.Error())
		}
	}

	e.logger.Info("cert enrollment successful")

	if err := e.getProfile(ctx, uid); err != nil {
		e.logger.Warn("error", "error", err.Error())
	}

	return &tls.Certificate{Certificate: [][]byte{cert.Raw}, PrivateKey: key, Leaf: cert}, ca, nil
}

func (e *Enroller) getProfile(ctx context.Context, uid string) error {
	body, err := NewRequest(e.client, e.getURL("/Marti/api/tls/profile/enrollment")).
		Auth(e.user, e.passwd).
		Args(map[string]string{"clientUID": uid}).
		Do(ctx)

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
	keyBytes, _ := rsa.GenerateKey(rand.Reader, keySize)

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

func LoadP12(filename, password string) (*tls.Certificate, []*x509.Certificate, error) {
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
