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
	event *cot.Event
	dat   []byte
}

const (
	lastSeenThreshold = time.Minute * 5
)

type App struct {
	Logger  *zap.SugaredLogger
	tcpport int
	udpport int
	webport int

	lat  float64
	lon  float64
	zoom int8

	clients map[string]*ClientHandler
	units   map[string]*model.Unit

	ctx     context.Context
	uid     string
	ch      chan *Msg
	logging bool

	clientMx sync.RWMutex
	unitMx   sync.RWMutex
}

func NewApp(tcpport, udpport, webport int, logger *zap.SugaredLogger) *App {
	return &App{
		Logger:   logger,
		tcpport:  tcpport,
		udpport:  udpport,
		webport:  webport,
		ch:       make(chan *Msg, 20),
		clients:  make(map[string]*ClientHandler, 0),
		units:    make(map[string]*model.Unit, 0),
		uid:      uuid.New().String(),
		clientMx: sync.RWMutex{},
		unitMx:   sync.RWMutex{},
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

func (app *App) AddClient(uid string, cl *ClientHandler) {
	app.clientMx.Lock()
	defer app.clientMx.Unlock()

	app.Logger.Infof("new client: %s", uid)
	app.clients[uid] = cl
}

func (app *App) RemoveClient(uid string) {
	app.clientMx.Lock()
	defer app.clientMx.Unlock()

	if _, ok := app.clients[uid]; ok {
		app.Logger.Infof("remove client: %s", uid)
		delete(app.clients, uid)
	}
}

func (app *App) AddUnit(uid string, u *model.Unit) {
	if u == nil {
		return
	}

	app.unitMx.Lock()
	defer app.unitMx.Unlock()

	app.units[uid] = u
}

func (app *App) RemoveUnit(uid string) {
	app.unitMx.Lock()
	defer app.unitMx.Unlock()

	if _, ok := app.units[uid]; ok {
		delete(app.units, uid)
	}
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
			app.SendTo(cot.MakePong(), uid)
			app.SendMsgToAll(msg.dat, uid)
			continue
		case msg.event.IsChat():
			app.Logger.Infof("chat %s %s", msg.event.Detail.Chat, msg.event.GetText())
		case strings.HasPrefix(msg.event.Type, "a-"):
			app.Logger.Debugf("pos %s (%s) stale %s", msg.event.Uid, msg.event.GetCallsign(), msg.event.Stale.Sub(time.Now()))
			app.AddUnit(msg.event.Uid, model.FromEvent(msg.event))
		case strings.HasPrefix(msg.event.Type, "b-"):
			app.Logger.Debugf("point %s (%s)", msg.event.Uid, msg.event.GetCallsign())
			app.AddUnit(msg.event.Uid, model.FromEvent(msg.event))
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
			app.cleanStale()
		}
	}
}

func (app *App) cleanStale() {
	app.unitMx.Lock()
	defer app.unitMx.Unlock()

	toDelete := make([]string, 0)
	for k, v := range app.units {
		if v.Evt.IsContact() {
			if v.LastSeen.Add(lastSeenThreshold).Before(time.Now()) {
				toDelete = append(toDelete, k)
				app.Logger.Debugf("removing contact %s (lastseen %s)", k, v.LastSeen.Sub(time.Now()))

			}
		} else {
			if v.Stale.Before(time.Now()) {
				toDelete = append(toDelete, k)
				app.Logger.Debugf("removing %s (stale %s)", k, v.Stale.Sub(time.Now()))
			}
		}
	}

	for _, uid := range toDelete {
		delete(app.units, uid)

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
	app.clientMx.RLock()
	defer app.clientMx.RUnlock()

	for uid, h := range app.clients {
		if uid != author {
			h.AddMsg(msg)
		}
	}
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
	app.clientMx.RLock()
	defer app.clientMx.RUnlock()

	if h, ok := app.clients[uid]; ok {
		h.AddMsg(msg)
	}
}

func (app *App) SendMsgToCallsign(msg []byte, callsign string) {
	app.clientMx.RLock()
	defer app.clientMx.RUnlock()

	for _, h := range app.clients {
		if h.Callsign == callsign {
			h.AddMsg(msg)
		}
	}
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
