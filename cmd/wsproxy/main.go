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

	"github.com/google/uuid"
	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/kdudkov/goatak/pkg/log"
	"github.com/kdudkov/goatak/pkg/tlsutil"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	selfPosSendPeriod = time.Second * 10
)

type Myself struct {
	UID      string
	Type     string
	Callsign string
	Team     string
	Role     string
	Lat      float64
	Lon      float64
	Alt      float64
	Ce       float64
	Le       float64
	Speed    float64
	Track    float64
	Device   string
	Platform string
	Os       string
	Version  string
}

type App struct {
	me           *Myself
	webPort      int
	webAddress   string
	mcastPort    int
	mcastAddress string
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
	mcastHandler *UdpClientHandler
	dumpServer   *os.File
	dumpMCast    *os.File
}

func (app *App) SetConnected(connected bool) {
	if connected {
		atomic.StoreUint32(&app.connected, 1)
	} else {
		atomic.StoreUint32(&app.connected, 0)
	}
}

// ProcessCotFromWSClient processes COT messages from websocket clients and forwards them to the server connection or multicast connection
func (app *App) ProcessCotFromWSClient(msg *cot.CotMessage) {
	if msg != nil {
		sent := false
		if app.connected == 1 {
			app.serverClient.SendMsg(msg)
			sent = true
		}
		if app.mcastHandler.IsActive() {
			app.mcastHandler.SendMsg(msg)
			sent = true
		}
		if !sent {
			app.logger.Info("not connected to server or multicast, drop message", slog.Any("msg", msg))
		}
	}
}

// ProcessCotFromMcast processes COT messages from multicast and forwards them to the websocket connected clients
func (app *App) ProcessCotFromMcast(msg *cot.CotMessage) {
	if len(app.wsClients) == 0 {
		app.logger.Info("no websocket clients connected, drop message", slog.Any("msg", msg))
		return
	}

	for _, ch := range app.wsClients {
		ch.SendMsg(msg)
	}

	buf, err := cot.MakeProtoPacket(msg.GetTakMessage())
	if err != nil {
		app.logger.Info("processCotFromMcast", slog.Any("error", err))
	}
	app.dumpMCast.Write(buf)
}

// ProcessCotFromTAKServer processes COT messages from the TAK server and forwards them to the websocket connected clients
func (app *App) ProcessCotFromTAKServer(msg *cot.CotMessage) {
	if len(app.wsClients) == 0 {
		app.logger.Info("no websocket clients connected, drop message", slog.Any("msg", msg))
		return
	}

	for _, ch := range app.wsClients {
		ch.SendMsg(msg)
	}

	buf, err := cot.MakeProtoPacket(msg.GetTakMessage())
	if err != nil {
		app.logger.Info("processCotFromTAKServer", slog.Any("error", err))
	}
	app.dumpServer.Write(buf)
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

	if app.mcastPort > 0 {
		go func() {
			addr := fmt.Sprintf("%s:%d", app.mcastAddress, app.mcastPort)
			app.logger.Info("listening multicast " + addr)

			app.mcastHandler = NewUdpClientHandler(app.logger, app.ProcessCotFromMcast)
			err := app.mcastHandler.Listen(addr)
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
		go app.myPosSender(ctx, wg)
		wg.Wait()
	}
}

func (app *App) myPosSender(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	app.sendMsg(app.makeMe())

	ticker := time.NewTicker(selfPosSendPeriod)
	defer ticker.Stop()

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			app.logger.Debug("sending pos")
			app.sendMsg(app.makeMe())
		}
	}
}

func (app *App) sendMsg(msg *cotproto.TakMessage) {
	if app.serverClient != nil {
		if app.serverClient.IsActive() {
			if err := app.serverClient.SendCot(msg); err != nil {
				app.logger.Error("error", slog.Any("error", err))
			}
		}
	}
	if app.mcastHandler != nil {
		if app.mcastHandler.IsActive() {
			if err := app.mcastHandler.SendCot(msg); err != nil {
				app.logger.Error("error", slog.Any("error", err))
			}
		}
	}
}

func (app *App) makeMe() *cotproto.TakMessage {
	ev := cot.BasicMsg(app.me.Type, app.me.UID, time.Minute*2)

	ev.CotEvent.Lat = app.me.Lat
	ev.CotEvent.Lon = app.me.Lon
	ev.CotEvent.Hae = app.me.Alt
	ev.CotEvent.Ce = app.me.Ce
	ev.CotEvent.Le = app.me.Le
	ev.CotEvent.How = "m-g"

	ev.CotEvent.Detail = &cotproto.Detail{
		Contact: &cotproto.Contact{
			Endpoint: "*:-1:stcp",
			Callsign: app.me.Callsign,
		},
		Group: &cotproto.Group{
			Name: app.me.Team,
			Role: app.me.Role,
		},
		Takv: &cotproto.Takv{
			Device:   app.me.Device,
			Platform: app.me.Platform,
			Os:       app.me.Os,
			Version:  app.me.Version,
		},
		Track: &cotproto.Track{
			Speed:  app.me.Speed,
			Course: app.me.Track,
		},
		PrecisionLocation: &cotproto.PrecisionLocation{
			Geopointsrc: "GPS",
			Altsrc:      "GPS",
		},
		Status: &cotproto.Status{Battery: 39},
	}

	ev.CotEvent.Detail.XmlDetail = fmt.Sprintf("<uid Droid=\"%s\"></uid>", app.me.Callsign)
	return ev
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

	_dumpServer, err := os.Create("dump_server.bin")
	if err != nil {
		logger.Error("Can't opend file for dump")
	}
	_dumpMCast, err := os.Create("dump_mcast.bin")
	if err != nil {
		logger.Error("Can't opend file for dump")
	}

	return &App{
		logger:      logger,
		host:        parts[0],
		tcpPort:     parts[1],
		tls:         tlsConn,
		dialTimeout: time.Second * 5,
		wsClients:   make(map[string]client.ClientHandler),
		dumpServer:  _dumpServer,
		dumpMCast:   _dumpMCast,
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
	k.Set("mcast_address", "239.2.3.1")
	k.Set("mcast_port", 6969)
	k.Set("ssl.password", "atakatak")
	k.Set("ssl.strict", false)

	k.Set("me.alt", 999999.0)
	k.Set("me.callsign", "wsproxy")
	k.Set("me.ce", 999999.0)
	k.Set("me.device", "wsproxy")
	k.Set("me.lat", 999999.0)
	k.Set("me.le", 999999.0)
	k.Set("me.lon", 999999.0)
	k.Set("me.platform", "GoATAK_client")
	k.Set("me.role", "HQ")
	k.Set("me.speed", 0.0)
	k.Set("me.team", "Blue")
	k.Set("me.track", 0.0)
	k.Set("me.type", "a-f-G-U-C")
	k.Set("me.uid", uuid.New().String())
	k.Set("me.version", "0.0.1")

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
	app.mcastPort = k.Int("mcast_port")
	app.mcastAddress = k.String("mcast_address")

	app.me = &Myself{
		Alt:      k.Float64("me.alt"),
		Callsign: k.String("me.callsign"),
		Ce:       k.Float64("me.ce"),
		Device:   k.String("me.device"),
		Lat:      k.Float64("me.lat"),
		Le:       k.Float64("me.le"),
		Lon:      k.Float64("me.lon"),
		Platform: k.String("me.platform"),
		Os:       k.String("me.os"),
		Role:     k.String("me.role"),
		Team:     k.String("me.team"),
		Type:     k.String("me.type"),
		UID:      k.String("me.uid"),
		Version:  k.String("me.version"),
	}

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

	app.dumpServer.Close()
	app.dumpMCast.Close()

	cancel()
}
