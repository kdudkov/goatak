package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kdudkov/goatak/cot"
	"github.com/spf13/viper"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kdudkov/goatak/cotproto"
	"github.com/kdudkov/goatak/model"
)

var (
	gitRevision            = "unknown"
	gitCommit              = "unknown"
	lastSeenOfflineTimeout = time.Minute * 2
)

type AppConfig struct {
	tcpPort int
	udpPort int
	webPort int
	apiPort int
	sslPort int

	logging  bool
	certFile string
	keyFile  string
}

type App struct {
	Logger         *zap.SugaredLogger
	packageManager *PackageManager
	config         *AppConfig
	lat            float64
	lon            float64
	zoom           int8

	handlers sync.Map
	units    sync.Map
	points   sync.Map
	messages []*model.ChatMessage

	ctx context.Context
	uid string
	ch  chan *cot.Msg
}

func NewApp(config *AppConfig, logger *zap.SugaredLogger) *App {
	return &App{
		Logger:         logger,
		config:         config,
		packageManager: NewPackageManager(logger),
		ch:             make(chan *cot.Msg, 20),
		handlers:       sync.Map{},
		units:          sync.Map{},
		points:         sync.Map{},
		uid:            uuid.New().String(),
	}
}

func (app *App) Run() {
	if err := app.packageManager.Init(); err != nil {
		log.Fatal(err)
	}

	var cancel context.CancelFunc

	app.ctx, cancel = context.WithCancel(context.Background())

	go func() {
		if err := app.ListenUDP(fmt.Sprintf(":%d", app.config.udpPort)); err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := app.ListenTCP(fmt.Sprintf(":%d", app.config.tcpPort)); err != nil {
			panic(err)
		}
	}()

	if app.config.keyFile != "" && app.config.certFile != "" {
		go func() {
			if err := app.ListenSSl(app.config.certFile, app.config.keyFile, fmt.Sprintf(":%d", app.config.sslPort)); err != nil {
				panic(err)
			}
		}()
	}

	NewHttp(app,
		fmt.Sprintf(":%d", app.config.webPort),
		fmt.Sprintf(":%d", app.config.apiPort),
	).Start()

	go app.EventProcessor()
	go app.cleaner()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-c
	app.Logger.Info("exiting...")
	cancel()
}

func (app *App) AddHandler(uid string, cl *ClientHandler) {
	app.Logger.Infof("new client: %s", uid)
	app.handlers.Store(uid, cl)
}

func (app *App) RemoveHandler(uid string) {
	if _, ok := app.handlers.Load(uid); ok {
		app.Logger.Infof("remove handler: %s", uid)
		app.handlers.Delete(uid)
	}
}

func (app *App) AddUnit(uid string, u *model.Unit) {
	if u == nil {
		return
	}
	app.units.Store(uid, u)
}

func (app *App) GetUnit(uid string) *model.Unit {
	if v, ok := app.units.Load(uid); ok {
		if unit, ok := v.(*model.Unit); ok {
			return unit
		} else {
			app.Logger.Errorf("invalid object for uid %s: %v", uid, v)
		}
	}
	return nil
}

func (app *App) RemoveUnit(uid string) {
	if _, ok := app.units.Load(uid); ok {
		app.units.Delete(uid)
	}
}

func (app *App) AddContact(uid string, u *model.Contact) {
	if u == nil {
		return
	}
	app.Logger.Infof("contact added %s", uid)
	app.units.Store(uid, u)

	callsing := u.GetCallsign()
	app.SendTo(uid, cot.MakeChatMessage(u.GetUID(), callsing, "Welcome"))

	app.units.Range(func(key, value interface{}) bool {
		switch v := value.(type) {
		case *model.Unit:
			app.SendTo(uid, v.GetMsg().TakMessage)
		case *model.Contact:
			if v.GetUID() != uid {
				app.SendTo(uid, v.GetMsg().TakMessage)
			}
		}
		return true
	})
}

func (app *App) GetContact(uid string) *model.Contact {
	if v, ok := app.units.Load(uid); ok {
		if contact, ok := v.(*model.Contact); ok {
			return contact
		} else {
			app.Logger.Errorf("invalid object for uid %s: %v", uid, v)
		}
	}
	return nil
}

func (app *App) AddPoint(uid string, p *model.Point) {
	if p == nil {
		return
	}
	app.points.Store(uid, p)
}

func (app *App) EventProcessor() {
	for msg := range app.ch {
		if msg.TakMessage.CotEvent == nil {
			continue
		}

		if app.config.logging {
			if f, err := os.OpenFile(msg.GetType()+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); err == nil {
				f.WriteString(msg.TakMessage.String())
				f.Close()
			} else {
				fmt.Println(err)
			}
		}

		switch {
		case msg.GetType() == "t-x-c-t":
			// ping
			app.Logger.Debugf("ping from %s", msg.GetUid())
			uid := msg.GetUid()
			if strings.HasSuffix(uid, "-ping") {
				uid = uid[:len(uid)-5]
			}
			if c := app.GetContact(uid); c != nil {
				c.SetLastSeenNow(nil)
			}
			app.SendTo(uid, cot.MakePong())
			app.SendToAllOther(msg.TakMessage, uid)
			continue
		case msg.IsChat():
			if c := model.MsgToChat(msg); c != nil {
				app.messages = append(app.messages, c)
			}
		case strings.HasPrefix(msg.GetType(), "a-"):
			app.Logger.Debugf("pos %s (%s) %s stale %s",
				msg.GetUid(),
				msg.GetCallsign(),
				msg.GetType(),
				msg.GetStale().Sub(time.Now()))
			if msg.IsContact() {
				if c := app.GetContact(msg.GetUid()); c != nil {
					c.SetLastSeenNow(msg)
				} else {
					app.AddContact(msg.GetUid(), model.ContactFromEvent(msg))
				}
			} else {
				app.AddUnit(msg.GetUid(), model.UnitFromEvent(msg))
			}
			// b-m-p-s-p-i digital pointer
			// b-m-p-s-m point
			// b-m-r route
		case strings.HasPrefix(msg.GetType(), "b-"):
			app.Logger.Debugf("point %s (%s) stale %s",
				msg.GetUid(),
				msg.GetCallsign(),
				msg.GetStale().Sub(time.Now()))
			app.AddPoint(msg.GetUid(), model.PointFromEvent(msg))
		default:
			app.Logger.Debugf("msg: %s", msg)
		}

		app.route(msg)
	}
}

func (app *App) route(msg *cot.Msg) {
	if len(msg.Detail.GetDest()) > 0 {
		for _, s := range msg.Detail.GetDest() {
			app.SendToCallsign(s, msg.TakMessage)
		}
	} else {
		app.SendToAllOther(msg.TakMessage, msg.From)
	}
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
				app.Logger.Debugf("removing %s", key)
			}

		case *model.Contact:
			if val.IsOld() {
				toDelete = append(toDelete, key.(string))
				app.Logger.Debugf("removing contact %s", key)
			} else {
				if val.IsOnline() && val.GetLastSeen().Add(lastSeenOfflineTimeout).Before(time.Now()) {
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

func (app *App) SendToAllOther(msg *cotproto.TakMessage, author string) {
	app.handlers.Range(func(key, value interface{}) bool {
		if key.(string) != author {
			if err := value.(*ClientHandler).AddMsg(msg); err != nil {
				app.Logger.Errorf("error sending to %s: %v", key, err)
			}
		}
		return true
	})
}

func (app *App) SendTo(uid string, msg *cotproto.TakMessage) {
	if h, ok := app.handlers.Load(uid); ok {
		if err := h.(*ClientHandler).AddMsg(msg); err != nil {
			app.Logger.Errorf("error sending to %s: %v", uid, err)
		}
	}
}

func (app *App) SendToCallsign(callsign string, msg *cotproto.TakMessage) {
	app.handlers.Range(func(key, value interface{}) bool {
		h := value.(*ClientHandler)
		if h.GetCallsign() == callsign {
			h.AddMsg(msg)
			return false
		}
		return true
	})
}

func main() {
	fmt.Printf("version %s %s\n", gitRevision, gitCommit)
	var logging = flag.Bool("logging", false, "save all events to files")
	var conf = flag.String("config", "goatak_server.yml", "name of config file")
	flag.Parse()

	viper.SetConfigFile(*conf)

	viper.SetDefault("web_port", 8080)
	viper.SetDefault("tcp_port", 8999)
	viper.SetDefault("udp_port", 8999)
	viper.SetDefault("ssl_port", 8089)
	viper.SetDefault("api_port", 8889)

	viper.SetDefault("me.lat", 35.462939)
	viper.SetDefault("me.lon", -97.537283)
	viper.SetDefault("me.zoom", 5)

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	flag.Parse()

	cfg := zap.NewDevelopmentConfig()
	logger, _ := cfg.Build()
	defer logger.Sync()

	config := &AppConfig{
		tcpPort:  viper.GetInt("tcp_port"),
		udpPort:  viper.GetInt("udp_port"),
		webPort:  viper.GetInt("web_port"),
		apiPort:  viper.GetInt("api_port"),
		sslPort:  viper.GetInt("ssl_port"),
		logging:  *logging,
		certFile: viper.GetString("cert_file"),
		keyFile:  viper.GetString("key_file"),
	}

	app := NewApp(config, logger.Sugar())

	app.lat = viper.GetFloat64("me.lat")
	app.lon = viper.GetFloat64("me.lon")
	app.zoom = int8(viper.GetInt("me.zoom"))
	app.Run()
}
