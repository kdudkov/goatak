package main

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/pkg/tlsutil"
)

const (
	alfaNum = "abcdefghijklmnopqrstuvwxyz012346789"
)

type App struct {
	g           *gocui.Gui
	ui          bool
	dialTimeout time.Duration
	host        string
	tcpPort     string
	webPort     int
	Logger      *zap.SugaredLogger
	tls         bool
	tlsCert     *tls.Certificate
	cas         *x509.CertPool
	remoteAPI   *RemoteAPI
	saveFile    string
	connected   uint32

	missions sync.Map

	callsign string
	uid      string
	typ      string
	team     string
	device   string
	version  string
	platform string
	os       string
	role     string
	zoom     int8

	cancel context.CancelFunc
}

func NewApp(uid string, callsign string, connectStr string, webPort int, logger *zap.SugaredLogger) *App {
	parts := strings.Split(connectStr, ":")

	if len(parts) != 3 {
		logger.Errorf("invalid connect string: %s", connectStr)

		return nil
	}

	var tlsConn bool

	switch parts[2] {
	case "tcp":
		tlsConn = false
	case "ssl":
		tlsConn = true
	default:
		logger.Errorf("invalid connect string: %s", connectStr)

		return nil
	}

	return &App{
		Logger:      logger,
		callsign:    callsign,
		uid:         uid,
		host:        parts[0],
		tcpPort:     parts[1],
		tls:         tlsConn,
		webPort:     webPort,
		dialTimeout: time.Second * 5,
		missions:    sync.Map{},
	}
}

func (app *App) Run(ctx context.Context) {
	var err error

	app.g, err = gocui.NewGui(gocui.OutputNormal)

	if err != nil {
		panic(err)
	}

	defer app.g.Close()

	app.g.SetManagerFunc(app.layout)

	if err := app.setBindings(); err != nil {
		panic(err)
	}

	app.remoteAPI = NewRemoteAPI(app.host, app.Logger)

	if app.tls {
		app.remoteAPI.SetTLS(app.getTLSConfig())
	}

	if m, err := app.remoteAPI.GetMissions(ctx); err == nil {
		for _, mm := range m {
			app.missions.Store(mm.Name, mm)
		}

		app.redraw()
	} else {
		panic(err)
	}

	if err := app.g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		app.Logger.Errorf(err.Error())
	}
}

func (app *App) stop(_ *gocui.Gui, _ *gocui.View) error {
	if app.cancel != nil {
		app.cancel()
	}

	return gocui.ErrQuit
}

func (app *App) SetConnected(connected bool) {
	if connected {
		atomic.StoreUint32(&app.connected, 1)
	} else {
		atomic.StoreUint32(&app.connected, 0)
	}

	app.redraw()
}

func (app *App) IsConnected() bool {
	return atomic.LoadUint32(&app.connected) != 0
}

func makeUID(callsign string) string {
	s := hex.EncodeToString(md5.New().Sum([]byte(callsign)))

	return "ANDROID-" + s[:16]
}

func (app *App) GetVersion() string {
	return fmt.Sprintf("%s %s", app.platform, app.version)
}

func RandString(strlen int) string {
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = alfaNum[rand.Intn(len(alfaNum))]
	}

	return string(result)
}

func (app *App) getTLSConfig() *tls.Config {
	conf := &tls.Config{ //nolint:exhaustruct
		Certificates: []tls.Certificate{*app.tlsCert},
		RootCAs:      app.cas,
		ClientCAs:    app.cas,
	}

	if !viper.GetBool("ssl.strict") {
		conf.InsecureSkipVerify = true
	}

	return conf
}

func main() {
	conf := flag.String("config", "goatak_client.yml", "name of config file")
	debug := flag.Bool("debug", false, "debug")
	saveFile := flag.String("file", "", "record all events to file")
	flag.Parse()

	viper.SetConfigFile(*conf)

	viper.SetDefault("server_address", "204.48.30.216:8087:tcp")
	viper.SetDefault("web_port", 8080)
	viper.SetDefault("me.callsign", RandString(10))
	viper.SetDefault("me.lat", 0.0)
	viper.SetDefault("me.lon", 0.0)
	viper.SetDefault("me.zoom", 5)
	viper.SetDefault("me.type", "a-f-G-U-C")
	viper.SetDefault("me.team", "Blue")
	viper.SetDefault("me.role", "HQ")
	viper.SetDefault("me.platform", "GoATAK_client")
	viper.SetDefault("me.version", getVersion())
	viper.SetDefault("ssl.password", "atakatak")
	viper.SetDefault("ssl.save_cert", true)
	viper.SetDefault("ssl.strict", false)

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	var cfg zap.Config
	if *debug {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "console"
	}

	//cfg.OutputPaths = []string{"webclient.log"}

	logger, _ := cfg.Build()
	defer logger.Sync()

	uid := viper.GetString("me.uid")
	if uid == "auto" || uid == "" {
		uid = makeUID(viper.GetString("me.callsign"))
	}

	app := NewApp(
		uid,
		viper.GetString("me.callsign"),
		viper.GetString("server_address"),
		viper.GetInt("web_port"),
		logger.Sugar(),
	)

	app.saveFile = *saveFile

	app.zoom = int8(viper.GetInt("me.zoom"))
	app.typ = viper.GetString("me.type")
	app.team = viper.GetString("me.team")
	app.role = viper.GetString("me.role")

	app.device = viper.GetString("me.device")
	app.version = viper.GetString("me.version")
	app.platform = viper.GetString("me.platform")
	app.os = viper.GetString("me.os")

	app.Logger.Infof("callsign: %s", app.callsign)
	app.Logger.Infof("uid: %s", app.uid)
	app.Logger.Infof("team: %s", app.team)
	app.Logger.Infof("role: %s", app.role)
	app.Logger.Infof("server: %s", viper.GetString("server_address"))

	ctx, cancel := context.WithCancel(context.Background())

	if app.tls {
		if user := viper.GetString("ssl.enroll_user"); user != "" {
			passw := viper.GetString("ssl.enroll_password")
			if passw == "" {
				fmt.Println("no enroll_password")

				return
			}

			enr := client.NewEnroller(app.Logger.Named("enroller"), app.host, user, passw, viper.GetBool("ssl.save_cert"))

			cert, cas, err := enr.GetOrEnrollCert(ctx, app.uid, app.GetVersion())
			if err != nil {
				app.Logger.Errorf("error while enroll cert: %s", err.Error())

				return
			}

			app.tlsCert = cert
			app.cas = tlsutil.MakeCertPool(cas...)
		} else {
			app.Logger.Infof("loading cert from file %s", viper.GetString("ssl.cert"))

			cert, cas, err := client.LoadP12(viper.GetString("ssl.cert"), viper.GetString("ssl.password"))
			if err != nil {
				app.Logger.Errorf("error while loading cert: %s", err.Error())

				return
			}

			tlsutil.LogCert(app.Logger, "loaded cert", cert.Leaf)
			app.tlsCert = cert
			app.cas = tlsutil.MakeCertPool(cas...)
		}
	}

	app.Run(ctx)
	cancel()
}
