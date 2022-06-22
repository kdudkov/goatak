package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/cotproto"
	"github.com/kdudkov/goatak/model"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	lastSeenOfflineTimeout = time.Minute * 10
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
	dialTimeout time.Duration
	addr        string
	webPort     int
	Logger      *zap.SugaredLogger
	ch          chan []byte
	units       sync.Map
	messages    []*model.ChatMessage
	tls         bool
	cl          *cot.ClientHandler

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
		addr:        fmt.Sprintf("%s:%s", parts[0], parts[1]),
		tls:         tlsConn,
		webPort:     webPort,
		units:       sync.Map{},
		dialTimeout: time.Second * 5,
	}
}

func (app *App) Run(ctx context.Context) {
	go func() {
		addr := fmt.Sprintf(":%d", app.webPort)
		app.Logger.Infof("listening %s", addr)
		if err := NewHttp(app, addr).Serve(); err != nil {
			panic(err)
		}
	}()

	go app.cleaner()

	app.ch = make(chan []byte, 20)

	for ctx.Err() == nil {
		conn, err := app.connect()
		if err != nil {
			app.Logger.Errorf("connect error: %s", err)
			time.Sleep(time.Second * 5)
			continue
		}

		app.Logger.Info("connected")
		wg := &sync.WaitGroup{}
		wg.Add(1)
		ctx1, cancel := context.WithCancel(ctx)

		app.cl = cot.NewClientHandler(app.addr, conn, &cot.HandlerConfig{
			Logger:    app.Logger,
			MessageCb: app.ProcessEvent,
			RemoveCb: func(ch *cot.ClientHandler) {
				wg.Done()
				cancel()
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

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-time.Tick(time.Minute):
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

func (app *App) ProcessEvent(msg *cot.Msg) {
	if c := app.GetContact(msg.GetUid()); c != nil {
		c.Update(nil)
	}

	switch {
	case msg.GetType() == "t-x-c-t":
		app.Logger.Debugf("ping from %s", msg.GetUid())
	case msg.GetType() == "t-x-d-d":
		app.removeByLink(msg)
	case msg.IsChat():
		if c := model.MsgToChat(msg); c != nil {
			if fromContact := app.GetContact(c.FromUid); fromContact != nil {
				c.From = fromContact.GetCallsign()
			}
			app.Logger.Infof("Chat %s (%s) -> %s (%s) \"%s\"", c.From, c.FromUid, c.To, c.ToUid, c.Text)
			app.messages = append(app.messages, c)
		}
	case strings.HasPrefix(msg.GetType(), "a-"):
		if msg.IsContact() {
			if msg.GetUid() == app.uid {
				app.Logger.Info("my own info")
				break
			}
			app.ProcessContact(msg)
		} else {
			app.ProcessUnit(msg)
		}
		return
	case strings.HasPrefix(msg.GetType(), "b-"):
		if uid, _ := msg.GetParent(); uid != app.uid {
			app.Logger.Infof("point %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
			app.AddPoint(msg.GetUid(), model.PointFromMsg(msg))
		} else {
			app.Logger.Infof("my own point %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
		}
	case strings.HasPrefix(msg.GetType(), "u-"):
		fmt.Println(msg.GetType())
	case msg.GetType() == "tak registration":
		app.Logger.Infof("registration %s %s", msg.GetUid(), msg.GetCallsign())
		return
	default:
		app.Logger.Debugf("unknown event: %s", msg.GetType())
	}
}

func (app *App) ProcessContact(msg *cot.Msg) {
	if c := app.GetContact(msg.GetUid()); c != nil {
		app.Logger.Infof("update contact %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
		c.Update(msg)
	} else {
		app.Logger.Infof("new contact %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
		if msg.GetUid() == app.uid {
			return
		}
		app.units.Store(msg.GetUid(), model.ContactFromMsg(msg))
	}
}

func (app *App) ProcessUnit(msg *cot.Msg) {
	if u := app.GetUnit(msg.GetUid()); u != nil {
		app.Logger.Infof("update unit %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
		u.Update(msg)
	} else {
		app.Logger.Infof("new unit %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
		app.units.Store(msg.GetUid(), model.UnitFromMsg(msg))
	}
}

func (app *App) AddPoint(uid string, u *model.Point) {
	if u == nil {
		return
	}
	app.units.Store(uid, u)
}

func (app *App) Remove(uid string) {
	if _, ok := app.units.Load(uid); ok {
		app.units.Delete(uid)
	}
}

func (app *App) AddContact(uid string, u *model.Contact) {
	if u == nil || uid == app.uid {
		return
	}
	app.units.Store(uid, u)
}

func (app *App) GetContact(uid string) *model.Contact {
	if v, ok := app.units.Load(uid); ok {
		if contact, ok := v.(*model.Contact); ok {
			return contact
		} else {
			app.Logger.Warnf("invalid object for uid %s: %s", uid, v)
		}
	}
	return nil
}

func (app *App) GetUnit(uid string) *model.Unit {
	if v, ok := app.units.Load(uid); ok {
		if contact, ok := v.(*model.Unit); ok {
			return contact
		} else {
			app.Logger.Warnf("invalid unit for uid %s: %s", uid, v)
		}
	}
	return nil
}

func (app *App) removeByLink(msg *cot.Msg) {
	if msg.Detail != nil && msg.Detail.HasChild("link") {
		uid := msg.Detail.GetFirstChild("link").GetAttr("uid")
		typ := msg.Detail.GetFirstChild("link").GetAttr("type")
		if uid == "" {
			app.Logger.Warnf("invalid remove message: %s", msg.Detail)
			return
		}
		if v, ok := app.units.Load(uid); ok {
			switch vv := v.(type) {
			case *model.Contact:
				app.Logger.Debugf("remove %s by message", uid)
				vv.SetOffline()
				return
			case *model.Unit, *model.Point:
				app.Logger.Debugf("remove unit/point %s type %s by message", uid, typ)
				//app.units.Delete(uid)
				return
			}
		}
	}
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
	}

	return ev
}

func RandString(strlen int) string {
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = alfaNum[rand.Intn(len(alfaNum))]
	}
	return string(result)
}

func (app *App) cleaner() {
	for {
		select {
		case <-time.Tick(time.Minute):
			app.cleanOldUnits()
		}
	}
}

func (app *App) cleanOldUnits() {
	toDelete := make([]string, 0)

	app.units.Range(func(key, value interface{}) bool {
		switch val := value.(type) {
		case *model.Unit:
			if val.IsOld() {
				toDelete = append(toDelete, key.(string))
				app.Logger.Debugf("removing unit %s", key)
			}
		case *model.Point:
			if val.IsOld() {
				toDelete = append(toDelete, key.(string))
				app.Logger.Debugf("removing point %s", key)
			}
		case *model.Contact:
			if val.IsOld() {
				toDelete = append(toDelete, key.(string))
				app.Logger.Debugf("removing contact %s", key)
			} else {
				if val.IsOnline() && val.GetStartTime().Add(lastSeenOfflineTimeout).Before(time.Now()) {
					val.SetOffline()
				}
			}
		}
		return true
	})

	for _, uid := range toDelete {
		app.units.Delete(uid)
	}
}

func (app *App) sendMyPoints() {
	app.units.Range(func(key, value interface{}) bool {
		switch val := value.(type) {
		case *model.Unit:
			if val.IsSend() {
				app.SendMsg(val.GetMsg().TakMessage)
			}
		case *model.Point:
			if val.IsSend() {
				app.SendMsg(val.GetMsg().TakMessage)
			}
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
	flag.Parse()

	viper.SetConfigFile(*conf)

	viper.SetDefault("server_address", "127.0.0.1:8089:tcp")
	viper.SetDefault("web_port", 8080)
	viper.SetDefault("me.callsign", RandString(10))
	viper.SetDefault("me.lat", 35.462939)
	viper.SetDefault("me.lon", -97.537283)
	viper.SetDefault("me.zoom", 5)
	viper.SetDefault("me.type", "a-f-G-U-C")
	viper.SetDefault("me.team", "Blue")
	viper.SetDefault("me.role", "HQ")
	viper.SetDefault("me.platform", "GoATAK_client")
	viper.SetDefault("me.version", fmt.Sprintf("%s:%s", gitBranch, gitRevision))
	viper.SetDefault("me.os", runtime.GOOS)
	viper.SetDefault("ssl.password", "atakatak")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cfg := zap.NewDevelopmentConfig()
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
	app.Logger.Infof("server: %s", app.addr)

	go app.Run(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-c
	cancel()
}
