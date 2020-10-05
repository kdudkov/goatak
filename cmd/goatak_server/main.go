package main

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/model"
)

var (
	gitRevision = "unknown"
	gitBranch   = "unknown"
)

type Msg struct {
	from  string
	event *cot.Event
	dat   []byte
}

type App struct {
	Logger  *zap.SugaredLogger
	tcpport int
	udpport int
	webport int

	lat  float64
	lon  float64
	zoom int8

	handlers sync.Map
	units    sync.Map

	ctx     context.Context
	uid     string
	ch      chan *Msg
	logging bool
}

func NewApp(tcpport, udpport, webport int, logger *zap.SugaredLogger) *App {
	return &App{
		Logger:   logger,
		tcpport:  tcpport,
		udpport:  udpport,
		webport:  webport,
		ch:       make(chan *Msg, 20),
		handlers: sync.Map{},
		units:    sync.Map{},
		uid:      uuid.New().String(),
	}
}

func (app *App) Run() {
	var cancel context.CancelFunc

	app.ctx, cancel = context.WithCancel(context.Background())

	go func() {
		if err := app.ListenUDP(fmt.Sprintf(":%d", app.udpport)); err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := app.ListenTCP(fmt.Sprintf(":%d", app.tcpport)); err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := NewHttp(app, fmt.Sprintf(":%d", app.webport)).Serve(); err != nil {
			panic(err)
		}
	}()

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

func (app *App) Remove(uid string) {
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

func (app *App) UpdateContact(uid string, f func(c *model.Contact) bool) bool {
	// we should never update event
	if v, ok := app.units.Load(uid); ok {
		if c, ok := v.(*model.Contact); ok {
			f(c)
		}
	}
	return false
}

func (app *App) EventProcessor() {
	for {
		msg := <-app.ch

		if msg.event == nil {
			continue
		}

		if app.logging {
			if f, err := os.OpenFile(msg.event.Type+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); err == nil {
				f.Write(msg.dat)
				f.Write([]byte{10})
				f.Close()
			} else {
				fmt.Println(err)
			}
		}

		switch {
		case msg.event.Type == "t-x-c-t":
			// ping
			app.Logger.Debugf("ping from %s", msg.event.Uid)
			uid := msg.event.Uid
			if strings.HasSuffix(uid, "-ping") {
				uid = uid[:len(uid)-5]
			}
			if c := app.GetContact(uid); c != nil {
				c.SetLastSeenNow(nil)
			}
			app.SendTo(cot.MakePong(), uid)
			app.SendMsgToAll(msg.dat, uid)
			continue
		case msg.event.IsChat():
			app.Logger.Infof("chat %s %s", msg.event.Detail.Chat, msg.event.GetText())
		case strings.HasPrefix(msg.event.Type, "a-"):
			app.Logger.Debugf("pos %s (%s) stale %s", msg.event.Uid, msg.event.GetCallsign(), msg.event.Stale.Sub(time.Now()))
			if msg.event.IsContact() {
				if c := app.GetContact(msg.event.Uid); c != nil {
					c.SetLastSeenNow(msg.event)
				} else {
					app.AddContact(msg.event.Uid, model.ContactFromEvent(msg.event))
				}
			} else {
				app.AddUnit(msg.event.Uid, model.UnitFromEvent(msg.event))
			}
		case strings.HasPrefix(msg.event.Type, "b-"):
			app.Logger.Debugf("point %s (%s) stale %s", msg.event.Uid, msg.event.GetCallsign(), msg.event.Stale.Sub(time.Now()))
			app.AddUnit(msg.event.Uid, model.UnitFromEvent(msg.event))
		default:
			app.Logger.Debugf("event: %s", msg.event)
		}

		app.route(msg)
	}
}

func (app *App) route(msg *Msg) {
	if len(msg.event.GetCallsignTo()) > 0 {
		for _, s := range msg.event.GetCallsignTo() {
			app.SendMsgToCallsign(msg.dat, s)
		}
	} else {
		app.SendMsgToAll(msg.dat, msg.event.Uid)
	}
}

func (app *App) cleaner() {
	ticker := time.NewTicker(time.Second * 120)

	for {
		select {
		case <-ticker.C:
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
			}
		}
		return true
	})

	for _, uid := range toDelete {
		app.units.Delete(uid)
	}
}

func (app *App) SendToAll(evt *cot.Event, author string) {
	msg, err := xml.Marshal(evt)

	if err != nil {
		app.Logger.Errorf("marshalling error: %v", err)
		return
	}

	app.SendMsgToAll(msg, author)
}

func (app *App) SendMsgToAll(msg []byte, author string) {
	app.handlers.Range(func(key, value interface{}) bool {
		if key.(string) != author {
			value.(*ClientHandler).AddMsg(msg)
		}
		return true
	})
}

func (app *App) SendTo(evt *cot.Event, uid string) {
	msg, err := xml.Marshal(evt)

	if err != nil {
		app.Logger.Errorf("marshalling error: %v", err)
		return
	}

	app.SendMsgTo(msg, uid)
}

func (app *App) SendToCallsign(evt *cot.Event, callsign string) {
	msg, err := xml.Marshal(evt)

	if err != nil {
		app.Logger.Errorf("marshalling error: %v", err)
		return
	}

	app.SendMsgToCallsign(msg, callsign)
}

func (app *App) SendMsgTo(msg []byte, uid string) {
	if h, ok := app.handlers.Load(uid); ok {
		h.(*ClientHandler).AddMsg(msg)
	}
}

func (app *App) SendMsgToCallsign(msg []byte, callsign string) {
	app.handlers.Range(func(key, value interface{}) bool {
		h := value.(*ClientHandler)
		if h.Callsign == callsign {
			h.AddMsg(msg)
			return false
		}
		return true
	})
}

func main() {
	fmt.Printf("version %s:%s\n", gitBranch, gitRevision)
	var logging = flag.Bool("logging", false, "save all events to files")
	var conf = flag.String("config", "goatak-server.yml", "name of config file")
	flag.Parse()

	viper.SetConfigFile(*conf)

	viper.SetDefault("web_port", 8080)
	viper.SetDefault("tcp_port", 8999)
	viper.SetDefault("udp_port", 8999)

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

	app := NewApp(
		viper.GetInt("tcp_port"),
		viper.GetInt("udp_port"),
		viper.GetInt("web_port"),
		logger.Sugar(),
	)
	app.logging = *logging
	app.lat = viper.GetFloat64("me.lat")
	app.lon = viper.GetFloat64("me.lon")
	app.zoom = int8(viper.GetInt("me.zoom"))
	app.Run()
}
