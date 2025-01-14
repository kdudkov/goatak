package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/log"
	"github.com/kdudkov/goatak/pkg/tlsutil"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type App struct {
	webPort      int
	webAddress   string
	dialTimeout  time.Duration
	host         string
	tcpPort      string
	logger       *slog.Logger
	tls          bool
	tlsStrict    bool
	tlsCert      *tls.Certificate
	cas          *x509.CertPool
	connected    uint32
	serverClient *client.ConnClientHandler
	wsClients    map[string]client.ClientHandler
}

func (app *App) SetConnected(connected bool) {
	if connected {
		atomic.StoreUint32(&app.connected, 1)
	} else {
		atomic.StoreUint32(&app.connected, 0)
	}
}

func (app *App) ProcessCotFromWSClient(msg *cot.CotMessage) {
	if msg != nil {
		if app.connected == 1 {
			app.serverClient.SendMsg(msg)
		} else {
			app.logger.Info("not connected to TAK server, drop message", slog.Any("msg", msg))
		}
	}
}

func (app *App) ProcessCotFromTAKServer(msg *cot.CotMessage) {
	app.logger.Info("process cot from server", slog.Any("msg", msg))

	if len(app.wsClients) == 0 {
		app.logger.Info("no websocket clients connected, drop message", slog.Any("msg", msg))
		return
	}

	for _, ch := range app.wsClients {
		ch.SendMsg(msg)
	}
}

func (app *App) ProcessRemoveFromTAKServer(ch client.ClientHandler) {
	app.logger.Info("process remove from server")
	app.SetConnected(false)
	//wg.Done()
	//cancel1()
	app.logger.Info("disconnected")
}

func (app *App) AddClientHandler(ch client.ClientHandler) {
	app.wsClients[ch.GetName()] = ch
}

func (app *App) RemoveClientHandler(name string) {
	delete(app.wsClients, name)
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

func (app *App) Init() {
}

// Run start client connection to the configured server. It loops until the context is canceled by signal or other means.
// Until running it will try to reconnect if the connection is lost.
func (app *App) Run(ctx context.Context) {
	if app.webPort >= 0 {
		go func() {
			addr := fmt.Sprintf("%s:%d", app.webAddress, app.webPort)
			app.logger.Info("listening " + addr)

			http := NewHttp(app)
			err := http.Listen(addr)
			if err != nil {
				panic(err)
			}

		}()
	}

	for ctx.Err() == nil {
		// Dial the server and connect to it.
		conn, err := app.connect()
		if err != nil {
			app.logger.Error("connect error", slog.Any("error", err))
			time.Sleep(time.Second * 5)

			continue
		}

		app.SetConnected(true)
		app.logger.Info("connected")
		app.logger.Info(fmt.Sprintf("conn: %+v", conn.RemoteAddr()))

		wg := new(sync.WaitGroup)
		wg.Add(1)

		//_, cancel1 := context.WithCancel(ctx)
		app.serverClient = client.NewConnClientHandler(fmt.Sprintf("%s:%s", app.host, app.tcpPort), conn, &client.HandlerConfig{
			MessageCb: app.ProcessCotFromTAKServer,
			RemoveCb: func(ch client.ClientHandler) {
				app.SetConnected(false)
				wg.Done()
				//cancel1()
				app.logger.Info("disconnected")
			},
			IsClient: true,
			UID:      "FIXME:UID:00001",
		})

		go app.serverClient.Start()
		wg.Wait()
	}
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
		logger.Error("invalid connect string " + connectStr)

		return nil
	}

	return &App{
		logger:      logger,
		host:        parts[0],
		tcpPort:     parts[1],
		tls:         tlsConn,
		dialTimeout: time.Second * 5,
		wsClients:   make(map[string]client.ClientHandler),
	}
}

func main() {
	conf := flag.String("config", "goatak_wsproxy.yml", "name of config file")
	debug := flag.Bool("debug", false, "debug")
	flag.Parse()

	k := koanf.New(".")
	k.Set("server_address", "204.48.30.216:8087:tcp")
	k.Set("web_address", "0.0.0.0")
	k.Set("web_port", 8088)
	k.Set("ssl.password", "atakatak")
	k.Set("ssl.strict", false)

	if err := k.Load(file.Provider(*conf), yaml.Parser()); err != nil {
		fmt.Printf("error loading config: %s", err.Error())
		return
	}

	var h slog.Handler
	if *debug {
		h = log.NewHandler(&slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		h = log.NewHandler(&slog.HandlerOptions{Level: slog.LevelInfo})
	}

	slog.SetDefault(slog.New(h))

	app := NewApp(k.String("server_address"))
	app.webPort = k.Int("web_port")
	app.webAddress = k.String("web_address")
	ctx, cancel := context.WithCancel(context.Background())

	if app.tls {
		app.logger.Info("loading cert from file " + k.String("ssl.cert"))
		cert, cas, err := client.LoadP12(k.String("ssl.cert"), k.String("ssl.password"))
		if err != nil {
			app.logger.Error("error while loading cert: " + err.Error())
			return
		}

		tlsutil.LogCert(app.logger, "loaded cert", cert.Leaf)
		app.tlsCert = cert
		app.cas = tlsutil.MakeCertPool(cas...)
	}

	app.Init()
	go app.Run(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	cancel()
}
