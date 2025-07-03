package config

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

	"github.com/kdudkov/goatak/internal/layers"
	"github.com/kdudkov/goatak/pkg/tlsutil"
)

type AppConfig struct {
	k *koanf.Koanf

	TlsCert    *tls.Certificate
	CertPool   *x509.CertPool
	ServerCert *x509.Certificate
	CA         []*x509.Certificate
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
		s1 := strings.ToLower(strings.TrimPrefix(s, prefix))
		for _, pr := range []string{"me_", "ssl_"} {
			if strings.HasPrefix(s1, pr) {
				slog.Info("ENV param: " + strings.Replace(s1, "_", ".", 1))
				return strings.Replace(s1, "_", ".", 1)
			}
		}
		slog.Info("ENV param: " + s1)

		return s1
	}), nil)
}

func (c *AppConfig) Bool(key string) bool {
	return c.k.Bool(key)
}

func (c *AppConfig) String(key string) string {
	return c.k.String(key)
}

func (c *AppConfig) FirstString(key ...string) string {
	for _, k := range key {
		if s := c.k.String(k); s != "" {
			return s
		}
	}

	return ""
}

func (c *AppConfig) Float64(key string) float64 {
	return c.k.Float64(key)
}

func (c *AppConfig) Int(key string) int {
	return c.k.Int(key)
}

func (c *AppConfig) Set(key string, v any) error {
	return c.k.Set(key, v)
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

func (c *AppConfig) MartiSSL() bool {
	return c.k.Bool("ssl.use_ssl") || c.k.Bool("ssl.marti")
}

func (c *AppConfig) EnrollSSL() bool {
	return c.k.Bool("ssl.enroll")
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

func (c *AppConfig) Layers() ([]*layers.LayerDescription, error) {
	if !c.k.Exists("layers") {
		return layers.GetDefaultLayers(), nil
	}

	res := make([]*layers.LayerDescription, 0)
	if err := c.k.Unmarshal("layers", &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *AppConfig) ProcessCerts() error {
	for _, name := range []string{"ssl.ca", "ssl.cert", "ssl.key"} {
		if c.k.String(name) == "" {
			slog.Info("no ssl config found (no value for " + name + ")")
			return nil
		}
	}

	ca, err := loadPem(c.k.String("ssl.ca"))
	if err != nil {
		return err
	}
	c.CA = ca

	certs, err := loadPem(c.k.String("ssl.cert"))
	if err != nil {
		return err
	}

	c.CertPool = tlsutil.MakeCertPool(ca...)
	for i, crt := range certs {
		c.CertPool.AddCert(crt)

		if i == 0 {
			c.ServerCert = crt
		} else {
			c.CA = append(c.CA, crt)
		}
	}

	tlsCert, err := tls.LoadX509KeyPair(c.k.String("ssl.cert"), c.k.String("ssl.key"))
	if err != nil {
		return err
	}

	c.TlsCert = &tlsCert

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
	k.Set("tls_addr", ":8089")
	k.Set("api_addr", ":8080")
	k.Set("local_addr", "localhost:8888")
	k.Set("data_dir", "data")

	k.Set("db", "db.sqlite")

	k.Set("delay", true)
	
	k.Set("me.lat", 59.8396)
	k.Set("me.lon", 31.0213)
	k.Set("users_file", "users.yml")

	k.Set("me.zoom", 10)
	k.Set("ssl.cert_ttl_days", 365)
}
