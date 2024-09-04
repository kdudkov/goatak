package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jroimartin/gocui"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/pkg/tlsutil"
)

type App struct {
	Logger      *slog.Logger
	g           *gocui.Gui
	dialTimeout time.Duration
	host        string
	tls         bool
	tlsCert     *tls.Certificate
	cas         *x509.CertPool
	remoteAPI   *RemoteAPI

	missions sync.Map

	cancel context.CancelFunc
}

func NewApp(connectStr string) *App {
	logger := slog.Default()

	parts := strings.Split(connectStr, ":")

	if len(parts) != 3 {
		logger.Error("invalid connect string: " + connectStr)

		return nil
	}

	var tlsConn bool

	switch parts[2] {
	case "tcp":
		tlsConn = false
	case "ssl":
		tlsConn = true
	default:
		logger.Error("invalid connect string: " + connectStr)

		return nil
	}

	return &App{
		Logger:      logger,
		host:        parts[0],
		tls:         tlsConn,
		dialTimeout: time.Second * 5,
		missions:    sync.Map{},
	}
}

func (app *App) Run(cmd string, args []string) {
	app.remoteAPI = NewRemoteAPI(app.host)

	if app.tls {
		app.remoteAPI.SetTLS(app.getTLSConfig(false))
	}

	switch cmd {
	case "files", "mp":
		app.listFiles()
	case "file", "get":
		if len(args) != 2 {
			fmt.Println("need hash and name")
			return
		}
		app.getFile(args[0], args[1])
	default:
		app.UI()
	}
}

func (app *App) listFiles() {
	res, err := app.remoteAPI.Search(context.Background())

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, p := range res {
		fmt.Printf("%s %s % -30s % -12s % -20s %s\n", p.UID, p.Hash, p.Name, p.Size, p.SubmissionUser, p.MIMEType)
	}
}

func (app *App) getFile(hash string, name string) {
	err := app.remoteAPI.GetFile(context.Background(), hash, func(r io.Reader) error {
		f, err := os.Create(name)

		if err != nil {
			return err
		}

		_, err = io.Copy(f, r)

		return err
	})

	if err != nil {
		fmt.Println(err)
	}
}

func (app *App) UI() {
	if m, err := app.remoteAPI.GetMissions(context.Background()); err == nil {
		for _, mm := range m {
			app.missions.Store(mm.Name, mm)
		}

		app.redraw()
	} else {
		panic(err)
	}

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

	if err := app.g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		app.Logger.Error("error", slog.Any("error", err))
	}
}

func (app *App) stop(_ *gocui.Gui, _ *gocui.View) error {
	if app.cancel != nil {
		app.cancel()
	}

	return gocui.ErrQuit
}

func (app *App) getTLSConfig(strict bool) *tls.Config {
	conf := &tls.Config{ //nolint:exhaustruct
		Certificates: []tls.Certificate{*app.tlsCert},
		RootCAs:      app.cas,
		ClientCAs:    app.cas,
	}

	if !strict {
		conf.InsecureSkipVerify = true
	}

	return conf
}

func main() {
	conf := flag.String("config", "goatak_client.yml", "name of config file")
	debug := flag.Bool("debug", false, "debug")
	cmd := flag.String("cmd", "", "command")
	flag.Parse()

	k := koanf.New(".")

	k.Set("server_address", "204.48.30.216:8087:tcp")
	k.Set("ssl.password", "atakatak")
	k.Set("ssl.save_cert", true)
	k.Set("ssl.strict", false)

	if err := k.Load(file.Provider(*conf), yaml.Parser()); err != nil {
		fmt.Printf("error loading config: %s", err.Error())
		return
	}

	var h slog.Handler
	if *debug {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	slog.SetDefault(slog.New(h))

	app := NewApp(k.String("server_address"))

	app.Logger.Info("server:" + k.String("server_address"))

	if app.tls {
		if user := k.String("ssl.enroll_user"); user != "" {
			passw := k.String("ssl.enroll_password")
			if passw == "" {
				fmt.Println("no enroll_password")

				return
			}

			enr := client.NewEnroller(app.host, user, passw, k.Bool("ssl.save_cert"), k.String("ssl.password"))

			cert, cas, err := enr.GetOrEnrollCert(context.Background(), uuid.NewString(), "")
			if err != nil {
				app.Logger.Error("error while enroll cert: " + err.Error())

				return
			}

			app.tlsCert = cert
			app.cas = tlsutil.MakeCertPool(cas...)
		} else {
			app.Logger.Info("loading cert from file " + k.String("ssl.cert"))

			cert, cas, err := client.LoadP12(k.String("ssl.cert"), k.String("ssl.password"))
			if err != nil {
				app.Logger.Error("error while loading cert: " + err.Error())

				return
			}

			tlsutil.LogCert(app.Logger, "loaded cert", cert.Leaf)
			app.tlsCert = cert
			app.cas = tlsutil.MakeCertPool(cas...)
		}
	}

	app.Run(*cmd, flag.Args())
}
