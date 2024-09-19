package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/knadh/koanf/v2"

	"github.com/kdudkov/goatak/pkg/tlsutil"
)

type AppConfig struct {
	k *koanf.Koanf

	tlsCert    *tls.Certificate
	certPool   *x509.CertPool
	serverCert *x509.Certificate
	ca         []*x509.Certificate
}

func (c *AppConfig) DataDir() string {
	return c.k.String("data_dir")
}

func (c *AppConfig) UsersFile() string {
	return c.k.String("users_file")
}

func (c *AppConfig) WelcomeMsg() string {
	return c.k.String("welcome_msg")
}

func (c *AppConfig) LogAll() bool {
	return c.k.Bool("log")
}
func (c *AppConfig) DataSync() bool {
	return c.k.Bool("datasync")
}
func (c *AppConfig) UseSSL() bool {
	return c.k.Bool("ssl.use_ssl")
}

func (c *AppConfig) CertTTLDays() int {
	return c.k.Int("ssl.cert_ttl_days")
}

func (c *AppConfig) Connections() []string {
	return nil
}

func (c *AppConfig) LogExclude() []string {
	return c.k.Strings("log_exclude")
}

func (c *AppConfig) BlacklistedUID() []string {
	return c.k.Strings("blacklist")
}

func (c *AppConfig) processCerts() error {
	for _, name := range []string{"ssl.ca", "ssl.cert", "ssl.key"} {
		if c.k.String(name) == "" {
			return nil
		}
	}

	roots := x509.NewCertPool()
	c.certPool = roots

	ca, err := loadPem(c.k.String("ssl.ca"))
	if err != nil {
		return err
	}

	for _, c := range ca {
		roots.AddCert(c)
	}

	c.ca = ca

	cert, err := loadPem(c.k.String("ssl.cert"))
	if err != nil {
		return err
	}

	if len(cert) > 0 {
		c.serverCert = cert[0]
	}

	for _, c := range cert {
		roots.AddCert(c)
	}

	tlsCert, err := tls.LoadX509KeyPair(c.k.String("ssl.cert"), c.k.String("ssl.key"))
	if err != nil {
		return err
	}

	c.tlsCert = &tlsCert

	return nil
}

func loadPem(name string) ([]*x509.Certificate, error) {
	if name == "" {
		return nil, nil
	}

	pemBytes, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("error loading %s: %s", name, err.Error())
	}

	return tlsutil.DecodeAllCerts(pemBytes)
}

func SetDefaults(k *koanf.Koanf) {
	k.Set("udp_addr", ":8999")
	k.Set("tcp_addr", ":8999")
	k.Set("ssl_addr", ":8089")
	k.Set("api_addr", ":8080")
	k.Set("data_dir", "data")

	k.Set("me.lat", 59.8396)
	k.Set("me.lon", 31.0213)
	k.Set("users_file", "users.yml")

	k.Set("me.zoom", 10)
	k.Set("ssl.cert_ttl_days", 365)
}
