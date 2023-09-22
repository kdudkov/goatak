package main

import (
	"context"
	"crypto/md5"
	"crypto/tls"
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

type Pos struct {
	lat float64
	lon float64
	mx  sync.RWMutex
}

type App struct {
	g           *gocui.Gui
	ui          bool
	dialTimeout time.Duration
	host        string
	tcpPort     string
	webPort     int
	Logger      *zap.SugaredLogger
	ch          chan []byte
	items       repository.ItemsRepository
	messages    *model.Messages
	tls         bool
	tlsCert     *tls.Certificate
	cl          *client.ConnClientHandler
	listeners   sync.Map
	textLogger  *TextLogger

	connected uint32

	callsign string
	uid      string
	typ      string
	team     string
	device   string
	version  string
	platform string
	os       string
	role     string
	pos      *Pos
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
		break
	case "ssl":
		tlsConn = true
		break
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
		items:       repository.NewItemsFileRepo(),
		dialTimeout: time.Second * 5,
		listeners:   sync.Map{},
		messages:    model.NewMessages(uid),
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

	app.ch = make(chan []byte, 20)
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
			Uid:      app.uid,
		})

		go app.cl.Start()
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

func makeUid(callsign string) string {
	h := md5.New()
	h.Write([]byte(callsign))
	uid := fmt.Sprintf("%x", h.Sum(nil))
	uid = uid[len(uid)-14:]

	return "ANDROID-" + uid
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
		if err := app.cl.SendMsg(msg); err != nil {
			app.Logger.Errorf("%v", err)
		}
	}
}

func (app *App) ProcessEvent(msg *cot.CotMessage) {
	app.textLogger.AddLine(fmt.Sprintf("%s %s (%s)", msg.GetType(), msg.GetUid(), msg.GetCallsign()))
	app.redraw()

	if c := app.items.Get(msg.GetUid()); c != nil {
		c.Update(nil)
	}

	switch {
	case msg.GetType() == "t-x-c-t":
		app.Logger.Debugf("ping from %s", msg.GetUid())
		if i := app.items.Get(msg.GetUid()); i != nil {
			i.SetOnline()
		}
	case msg.GetType() == "t-x-c-t-r":
		app.Logger.Debugf("pong from %s", msg.GetUid())
		if i := app.items.Get(msg.GetUid()); i != nil {
			i.SetOnline()
		}
	case msg.GetType() == "t-x-d-d":
		app.removeByLink(msg)
	case msg.IsChat():
		//fmt.Println(msg.TakMessage.GetCotEvent().Detail.XmlDetail)
		if c := model.MsgToChat(msg); c != nil {
			if c.From == "" {
				c.From = app.GetCallsign(c.FromUid)
			}
			app.Logger.Infof("%s", c)
			app.messages.Add(c)
		}
		break
	case msg.IsChatReceipt():
		app.Logger.Infof("got receipt %s", msg.GetType())
		break
	case strings.HasPrefix(msg.GetType(), "a-"):
		if msg.GetUid() == app.uid {
			break
		}
		app.ProcessItem(msg)
	case strings.HasPrefix(msg.GetType(), "b-"):
		if uid, _ := msg.GetParent(); uid != app.uid {
			app.Logger.Debugf("point %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
			app.ProcessItem(msg)
		} else {
			app.Logger.Debugf("my own point %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
		}
	case strings.HasPrefix(msg.GetType(), "u-"):
		fmt.Println(msg.GetType())
	case msg.GetType() == "tak registration":
		app.Logger.Debugf("registration %s %s", msg.GetUid(), msg.GetCallsign())
	default:
		app.Logger.Debugf("unknown event: %s", msg.GetType())
	}
}

func (app *App) ProcessItem(msg *cot.CotMessage) {
	cl := model.GetClass(msg)
	if cl == model.CONTACT {
		app.messages.CheckCallsing(msg.GetUid(), msg.GetCallsign())
	}
	if c := app.items.Get(msg.GetUid()); c != nil {
		app.Logger.Debugf("update %s %s (%s) %s", cl, msg.GetUid(), msg.GetCallsign(), msg.GetType())
		c.Update(msg)
	} else {
		app.Logger.Infof("new %s %s (%s) %s", cl, msg.GetUid(), msg.GetCallsign(), msg.GetType())
		app.items.Store(model.FromMsg(msg))
	}
}

func (app *App) GetCallsign(uid string) string {
	i := app.items.Get(uid)
	if i != nil {
		return i.GetCallsign()
	}
	return uid
}

func (app *App) removeByLink(msg *cot.CotMessage) {
	if msg.Detail != nil && msg.Detail.Has("link") {
		uid := msg.Detail.GetFirst("link").GetAttr("uid")
		typ := msg.Detail.GetFirst("link").GetAttr("type")
		if uid == "" {
			app.Logger.Warnf("invalid remove message: %s", msg.Detail)
			return
		}
		if v := app.items.Get(uid); v != nil {
			switch v.GetClass() {
			case model.CONTACT:
				app.Logger.Debugf("remove %s by message", uid)
				v.SetOffline()
				app.processChange(v)
				return
			case model.UNIT, model.POINT:
				app.Logger.Debugf("remove unit/point %s type %s by message", uid, typ)
				//app.units.Delete(uid)
				return
			}
		}
	}
}

func (app *App) processChange(u *model.Item) {

}

func (app *App) MakeMe() *cotproto.TakMessage {
	ev := cot.BasicMsg(app.typ, app.uid, time.Minute*2)
	lat, lon := app.pos.Get()
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
			} else {
				if item.IsOnline() && item.GetLastSeen().Add(lastSeenOfflineTimeout).Before(time.Now()) {
					item.SetOffline()
				}
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

func NewPos(lat, lon float64) *Pos {
	return &Pos{lon: lon, lat: lat, mx: sync.RWMutex{}}
}

func (p *Pos) Set(lat, lon float64) {
	if p == nil {
		return
	}
	p.mx.Lock()
	defer p.mx.Unlock()
	p.lat = lat
	p.lon = lon
}

func (p *Pos) Get() (float64, float64) {
	if p == nil {
		return 0, 0
	}
	p.mx.RLock()
	defer p.mx.RUnlock()
	return p.lat, p.lon
}

func main() {
	var conf = flag.String("config", "goatak_client.yml", "name of config file")
	var noweb = flag.Bool("noweb", false, "do not start web server")
	var ui = flag.Bool("ui", false, "do not start web server")
	var debug = flag.Bool("debug", false, "debug")
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
	viper.SetDefault("me.version", fmt.Sprintf("%s:%s", gitBranch, gitRevision))
	//viper.SetDefault("me.os", runtime.GOOS)
	viper.SetDefault("ssl.password", "atakatak")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
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
		uid = makeUid(viper.GetString("me.callsign"))
	}

	app := NewApp(
		uid,
		viper.GetString("me.callsign"),
		viper.GetString("server_address"),
		viper.GetInt("web_port"),
		logger.Sugar(),
	)

	app.ui = *ui

	if *noweb {
		app.webPort = 0
	}

	app.pos = NewPos(viper.GetFloat64("me.lat"), viper.GetFloat64("me.lon"))
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

			enr := NewEnroller(app.host, user, passw)
			if app.tlsCert, err = enr.enrollCert(app.uid, app.GetVersion()); err != nil {
				app.Logger.Errorf("error while enroll cert: %s", err.Error())
				return
			}
			app.Logger.Infof("cert enrollment successful")
		} else {
			app.loadCerts()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	app.Init(cancel)
	go app.Run(ctx)

	if app.ui {
		if err := app.g.MainLoop(); err != nil && err != gocui.ErrQuit {
			app.Logger.Errorf(err.Error())
		}
	} else {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
	}
	cancel()
}
