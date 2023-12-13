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
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/internal/repository"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/kdudkov/goatak/pkg/model"
	"github.com/kdudkov/goatak/pkg/tlsutil"
)

const (
	selfPosSendPeriod      = time.Minute
	lastSeenOfflineTimeout = time.Minute * 15
	alfaNum                = "abcdefghijklmnopqrstuvwxyz012346789"
)

var (
	gitRevision = "unknown"
	gitBranch   = "unknown"
)

type App struct {
	g               *gocui.Gui
	ui              bool
	dialTimeout     time.Duration
	host            string
	tcpPort         string
	webPort         int
	Logger          *zap.SugaredLogger
	ch              chan []byte
	items           repository.ItemsRepository
	messages        *model.Messages
	tls             bool
	tlsCert         *tls.Certificate
	cas             *x509.CertPool
	cl              *client.ConnClientHandler
	listeners       sync.Map
	textLogger      *TextLogger
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
		Logger:          logger,
		callsign:        callsign,
		uid:             uid,
		host:            parts[0],
		tcpPort:         parts[1],
		tls:             tlsConn,
		webPort:         webPort,
		items:           repository.NewItemsMemoryRepo(),
		dialTimeout:     time.Second * 5,
		listeners:       sync.Map{},
		messages:        model.NewMessages(uid),
		eventProcessors: make([]*EventProcessor, 0),
		pos:             atomic.Pointer[model.Pos]{},
	}
}

func (app *App) Init(cancel context.CancelFunc) {
	if app.ui {
		var err error

		app.g, err = gocui.NewGui(gocui.OutputNormal)
		if err != nil {
			panic(err)
		}

		app.g.SetManagerFunc(app.layout)

		if err := app.g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone,
			func(_ *gocui.Gui, _ *gocui.View) error { cancel(); return gocui.ErrQuit }); err != nil {
			panic(err)
		}

		app.textLogger = NewTextLogger()
	}

	app.remoteAPI = NewRemoteAPI(app.host)

	if app.tls {
		app.remoteAPI.SetTLS(app.getTLSConfig())
	}

	app.ch = make(chan []byte, 20)
	app.InitMessageProcessors()
}

func (app *App) Run(ctx context.Context) {
	if app.ui {
		defer app.g.Close()
	}

	if app.webPort != 0 {
		go func() {
			addr := fmt.Sprintf(":%d", app.webPort)
			app.Logger.Infof("listening %s", addr)

			if err := NewHttp(app, addr).Serve(); err != nil {
				panic(err)
			}
		}()
	}

	go app.cleaner()

	for ctx.Err() == nil {
		conn, err := app.connect()
		if err != nil {
			app.Logger.Errorf("connect error: %s", err)
			time.Sleep(time.Second * 5)

			continue
		}

		app.SetConnected(true)
		app.Logger.Info("connected")

		wg := new(sync.WaitGroup)
		wg.Add(1)

		ctx1, cancel1 := context.WithCancel(ctx)

		app.cl = client.NewConnClientHandler(fmt.Sprintf("%s:%s", app.host, app.tcpPort), conn, &client.HandlerConfig{
			Logger:    app.Logger,
			MessageCb: app.ProcessEvent,
			RemoveCb: func(ch client.ClientHandler) {
				app.SetConnected(false)
				wg.Done()
				cancel1()
				app.Logger.Info("disconnected")
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

	app.redraw()
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
			app.Logger.Debugf("sending pos")
			app.SendMsg(app.MakeMe())
			app.sendMyPoints()
		}
	}
}

func (app *App) SendMsg(msg *cotproto.TakMessage) {
	if app.cl != nil {
		if err := app.cl.SendCot(msg); err != nil {
			app.Logger.Errorf("%v", err)
		}
	}
}

func (app *App) ProcessEvent(msg *cot.CotMessage) {
	for _, prc := range app.eventProcessors {
		if cot.MatchAnyPattern(msg.GetType(), prc.include...) {
			app.Logger.Debugf("msg is processed by %s", prc.name)
			prc.cb(msg)
		}
	}
}

func (app *App) processChange(u *model.Item) {
}

func (app *App) MakeMe() *cotproto.TakMessage {
	ev := cot.BasicMsg(app.typ, app.uid, time.Minute*2)
	lat, lon := app.pos.Load().Get()
	ev.CotEvent.Lat = lat
	ev.CotEvent.Lon = lon
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
			Speed:  0,
			Course: 0,
		},
		PrecisionLocation: &cotproto.PrecisionLocation{
			Geopointsrc: "",
			Altsrc:      "DTED2",
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

func (app *App) cleaner() {
	for range time.Tick(time.Minute) {
		app.cleanOldUnits()
	}
}

func (app *App) cleanOldUnits() {
	toDelete := make([]string, 0)

	app.items.ForEach(func(item *model.Item) bool {
		switch item.GetClass() {
		case model.UNIT, model.POINT:
			if item.IsOld() {
				toDelete = append(toDelete, item.GetUID())
				app.Logger.Debugf("removing %s %s", item.GetClass(), item.GetUID())
			}
		case model.CONTACT:
			if item.IsOld() {
				toDelete = append(toDelete, item.GetUID())
				app.Logger.Debugf("removing contact %s", item.GetUID())
			} else if item.IsOnline() && item.GetLastSeen().Add(lastSeenOfflineTimeout).Before(time.Now()) {
				item.SetOffline()
			}
		}

		return true
	})

	for _, uid := range toDelete {
		app.items.Remove(uid)
	}
}

func (app *App) sendMyPoints() {
	app.items.ForEach(func(item *model.Item) bool {
		if item.IsSend() {
			app.SendMsg(item.GetMsg().TakMessage)
		}

		return true
	})
}

func getVersion() string {
	if gitBranch != "master" && gitBranch != "unknowm" {
		return fmt.Sprintf("%s:%s", gitBranch, gitRevision)
	}

	return gitRevision
}

func main() {
	conf := flag.String("config", "goatak_client.yml", "name of config file")
	noweb := flag.Bool("noweb", false, "do not start web server")
	ui := flag.Bool("ui", false, "do not start web server")
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

	if *ui {
		cfg.OutputPaths = []string{"webclient.log"}
	}

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

	app.ui = *ui
	app.saveFile = *saveFile

	if *noweb {
		app.webPort = 0
	}

	app.pos.Store(model.NewPos(viper.GetFloat64("me.lat"), viper.GetFloat64("me.lon")))
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

	if app.tls {
		if user := viper.GetString("ssl.enroll_user"); user != "" {
			passw := viper.GetString("ssl.enroll_password")
			if passw == "" {
				fmt.Println("no enroll_password")

				return
			}

			enr := NewEnroller(app.Logger.Named("enroller"), app.host, user, passw, viper.GetBool("ssl.save_cert"))

			cert, cas, err := enr.getOrEnrollCert(app.uid, app.GetVersion())
			if err != nil {
				app.Logger.Errorf("error while enroll cert: %s", err.Error())

				return
			}

			app.tlsCert = cert
			app.cas = tlsutil.MakeCertPool(cas...)
		} else {
			app.Logger.Infof("loading cert from file %s", viper.GetString("ssl.cert"))

			cert, cas, err := loadP12(viper.GetString("ssl.cert"), viper.GetString("ssl.password"))
			if err != nil {
				app.Logger.Errorf("error while loading cert: %s", err.Error())

				return
			}

			tlsutil.LogCert(app.Logger, "loaded cert", cert.Leaf)
			app.tlsCert = cert
			app.cas = tlsutil.MakeCertPool(cas...)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	app.Init(cancel)

	go app.Run(ctx)

	if app.ui {
		if err := app.g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
			app.Logger.Errorf(err.Error())
		}
	} else {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
	}

	cancel()
}
