package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
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

func NewAppConfig() *AppConfig {
	c := &AppConfig{k: koanf.New(".")}

	setDefaults(c.k)

	return c
}

func (c *AppConfig) Load(filename ...string) bool {
	loaded := false

	for _, name := range filename {
		if err := c.k.Load(file.Provider(name), yaml.Parser()); err != nil {
			slog.Info(fmt.Sprintf("error loading config: %s", err.Error()))
		} else {
			loaded = true
		}
	}

	return loaded
}

func (c *AppConfig) LoadEnv(prefix string) error {
	return c.k.Load(env.Provider(prefix, ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, prefix)), "_", ".", -1)
	}), nil)
}

func (c *AppConfig) Bool(key string) bool {
	return c.k.Bool(key)
}

func (c *AppConfig) String(key string) string {
	return c.k.String(key)
}

func (c *AppConfig) Float64(key string) float64 {
	return c.k.Float64(key)
}

func (c *AppConfig) Int(key string) int {
	return c.k.Int(key)
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

	ca, err := loadPem(c.k.String("ssl.ca"))
	if err != nil {
		return err
	}

	c.certPool = tlsutil.MakeCertPool(ca...)
	c.ca = ca

	cert, err := loadPem(c.k.String("ssl.cert"))
	if err != nil {
		return err
	}

	if len(cert) > 0 {
		c.serverCert = cert[0]
	}

	for _, crt := range cert {
		c.certPool.AddCert(crt)
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

func setDefaults(k *koanf.Koanf) {
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
