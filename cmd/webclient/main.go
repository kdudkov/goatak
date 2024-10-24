package main

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kdudkov/goatak/pkg/gpsd"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/kdudkov/goutils/callback"

	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/internal/repository"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/kdudkov/goatak/pkg/log"
	"github.com/kdudkov/goatak/pkg/model"
	"github.com/kdudkov/goatak/pkg/tlsutil"
)

const (
	selfPosSendPeriod      = time.Minute
	lastSeenOfflineTimeout = time.Minute * 15
	alfaNum                = "abcdefghijklmnopqrstuvwxyz012346789"
)

type App struct {
	dialTimeout     time.Duration
	host            string
	tcpPort         string
	webPort         int
	gpsd            string
	logger          *slog.Logger
	ch              chan []byte
	items           repository.ItemsRepository
	chatMessages    *model.ChatMessages
	tls             bool
	tlsStrict       bool
	tlsCert         *tls.Certificate
	cas             *x509.CertPool
	cl              *client.ConnClientHandler
	chatCb          *callback.Callback[*model.ChatMessage]
	eventProcessors []*EventProcessor
	remoteAPI       *RemoteAPI
	saveFile        string
	connected       uint32

	callsign string
	uid      string
	typ      string
	team     string
	device   string
	version  string
	platform string
	os       string
	role     string
	pos      atomic.Pointer[model.Pos]
	zoom     int8
}

func NewApp(uid string, callsign string, connectStr string, webPort int) *App {
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
		logger:          logger,
		callsign:        callsign,
		uid:             uid,
		host:            parts[0],
		tcpPort:         parts[1],
		tls:             tlsConn,
		webPort:         webPort,
		items:           repository.NewItemsMemoryRepo(time.Minute * 5),
		dialTimeout:     time.Second * 5,
		chatCb:          callback.New[*model.ChatMessage](),
		chatMessages:    model.NewChatMessages(uid),
		eventProcessors: make([]*EventProcessor, 0),
		pos:             atomic.Pointer[model.Pos]{},
	}
}

func (app *App) Init() {
	app.remoteAPI = NewRemoteAPI(app.host, app.logger.With("logger", "api"))

	if app.tls {
		app.remoteAPI.SetTLS(app.getTLSConfig(app.tlsStrict))
	}

	app.ch = make(chan []byte, 20)
	app.InitMessageProcessors()
}

func (app *App) Run(ctx context.Context) {
	_ = app.items.Start()

	if app.webPort >= 0 {
		go func() {
			addr := fmt.Sprintf(":%d", app.webPort)
			app.logger.Info("listening " + addr)

			if err := NewHttp(app).Listen(addr); err != nil {
				panic(err)
			}
		}()
	}

	if app.gpsd != "" {
		c := gpsd.New(app.gpsd, app.logger.With("logger", "gpsd"))
		go c.Listen(ctx, func(lat, lon, alt, speed, track float64) {
			app.pos.Store(model.NewPosFull(lat, lon, alt, speed, track))
		})
	}

	for ctx.Err() == nil {
		conn, err := app.connect()
		if err != nil {
			app.logger.Error("connect error", slog.Any("error", err))
			time.Sleep(time.Second * 5)

			continue
		}

		app.SetConnected(true)
		app.logger.Info("connected")

		wg := new(sync.WaitGroup)
		wg.Add(1)

		ctx1, cancel1 := context.WithCancel(ctx)

		app.cl = client.NewConnClientHandler(fmt.Sprintf("%s:%s", app.host, app.tcpPort), conn, &client.HandlerConfig{
			MessageCb: app.ProcessEvent,
			RemoveCb: func(ch client.ClientHandler) {
				app.SetConnected(false)
				wg.Done()
				cancel1()
				app.logger.Info("disconnected")
			},
			IsClient: true,
			UID:      app.uid,
		})

		go app.cl.Start()
		go app.periodicGetter(ctx1)
		go app.myPosSender(ctx1, wg)

		wg.Wait()
	}
}

func (app *App) SetConnected(connected bool) {
	if connected {
		atomic.StoreUint32(&app.connected, 1)
	} else {
		atomic.StoreUint32(&app.connected, 0)
	}
}

func (app *App) IsConnected() bool {
	return atomic.LoadUint32(&app.connected) != 0
}

func makeUID(callsign string) string {
	s := hex.EncodeToString(md5.New().Sum([]byte(callsign)))

	return "ANDROID-" + s[:16]
}

func (app *App) myPosSender(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	app.SendMsg(app.MakeMe())

	ticker := time.NewTicker(selfPosSendPeriod)
	defer ticker.Stop()

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			app.logger.Debug("sending pos")
			app.SendMsg(app.MakeMe())
			app.sendMyPoints()
		}
	}
}

func (app *App) SendMsg(msg *cotproto.TakMessage) {
	if app.cl != nil {
		if err := app.cl.SendCot(msg); err != nil {
			app.logger.Error("error", slog.Any("error", err))
		}
	}
}

func (app *App) ProcessEvent(msg *cot.CotMessage) {
	for _, prc := range app.eventProcessors {
		if cot.MatchAnyPattern(msg.GetType(), prc.include...) {
			app.logger.Debug("msg is processed by " + prc.name)
			prc.cb(msg)
		}
	}
}

func (app *App) MakeMe() *cotproto.TakMessage {
	ev := cot.BasicMsg(app.typ, app.uid, time.Minute*2)
	pos := app.pos.Load()

	ev.CotEvent.Lat = pos.GetLat()
	ev.CotEvent.Lon = pos.GetLon()
	ev.CotEvent.Hae = pos.GetAlt()
	ev.CotEvent.Ce = pos.GetCe()

	ev.CotEvent.Detail = &cotproto.Detail{
		Contact: &cotproto.Contact{
			Endpoint: "*:-1:stcp",
			Callsign: app.callsign,
		},
		Group: &cotproto.Group{
			Name: app.team,
			Role: app.role,
		},
		Takv: &cotproto.Takv{
			Device:   app.device,
			Platform: app.platform,
			Os:       app.os,
			Version:  app.version,
		},
		Track: &cotproto.Track{
			Speed:  pos.GetSpeed(),
			Course: pos.GetTrack(),
		},
		PrecisionLocation: &cotproto.PrecisionLocation{
			Geopointsrc: "GPS",
			Altsrc:      "GPS",
		},
		Status: &cotproto.Status{Battery: 39},
	}
	ev.CotEvent.Detail.XmlDetail = fmt.Sprintf("<uid Droid=\"%s\"></uid>", app.callsign)

	return ev
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

func (app *App) sendMyPoints() {
	app.items.ForEach(func(item *model.Item) bool {
		if item.IsSend() {
			app.SendMsg(item.GetMsg().GetTakMessage())
		}

		return true
	})
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
	noweb := flag.Bool("noweb", false, "do not start web server")
	debug := flag.Bool("debug", false, "debug")
	saveFile := flag.String("file", "", "record all events to file")
	flag.Parse()

	k := koanf.New(".")

	k.Set("server_address", "204.48.30.216:8087:tcp")
	k.Set("web_port", 8080)
	k.Set("me.callsign", RandString(10))
	k.Set("me.lat", 0.0)
	k.Set("me.lon", 0.0)
	k.Set("me.zoom", 5)
	k.Set("me.type", "a-f-G-U-C")
	k.Set("me.team", "Blue")
	k.Set("me.role", "HQ")
	k.Set("me.platform", "GoATAK_client")
	k.Set("me.version", getVersion())
	k.Set("ssl.password", "atakatak")
	k.Set("ssl.save_cert", true)
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

	uid := k.String("me.uid")
	if uid == "auto" || uid == "" {
		uid = makeUID(k.String("me.callsign"))
	}

	app := NewApp(
		uid,
		k.String("me.callsign"),
		k.String("server_address"),
		k.Int("web_port"),
	)

	app.saveFile = *saveFile

	if *noweb {
		app.webPort = -1
	}

	app.pos.Store(model.NewPos(k.Float64("me.lat"), k.Float64("me.lon")))
	app.zoom = int8(k.Int("me.zoom"))
	app.typ = k.String("me.type")
	app.team = k.String("me.team")
	app.role = k.String("me.role")

	app.device = k.String("me.device")
	app.version = k.String("me.version")
	app.platform = k.String("me.platform")
	app.os = k.String("me.os")

	app.gpsd = k.String("gpsd")

	app.logger.Info("callsign: " + app.callsign)
	app.logger.Info("uid: " + app.uid)
	app.logger.Info("team: " + app.team)
	app.logger.Info("role: " + app.role)
	app.logger.Info("server: " + k.String("server_address"))

	ctx, cancel := context.WithCancel(context.Background())

	if app.tls {
		if user := k.String("ssl.enroll_user"); user != "" {
			passw := k.String("ssl.enroll_password")
			if passw == "" {
				fmt.Println("no enroll_password")

				return
			}

			enr := client.NewEnroller(app.host, user, passw, k.Bool("ssl.save_cert"), k.String("ssl.password"))

			cert, cas, err := enr.GetOrEnrollCert(ctx, app.uid, app.GetVersion())
			if err != nil {
				app.logger.Error("error while enroll cert: " + err.Error())

				return
			}

			app.tlsCert = cert
			app.cas = tlsutil.MakeCertPool(cas...)
		} else {
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
	}

	app.Init()

	go app.Run(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	cancel()
}
