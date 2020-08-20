package main

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"gotac/cot"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	gitRevision = "unknown"
	gitBranch   = "unknown"
)

type App struct {
	Logger *zap.SugaredLogger
	port   int

	clients map[string]*ClientHandler
	points  map[string]*cot.Point
	mx      sync.RWMutex
	ctx     context.Context
	uid     string
	ch      chan *cot.Event
}

func NewApp(port int, logger *zap.SugaredLogger) *App {
	return &App{
		Logger:  logger,
		port:    port,
		mx:      sync.RWMutex{},
		ch:      make(chan *cot.Event, 20),
		clients: make(map[string]*ClientHandler, 0),
		points:  make(map[string]*cot.Point, 0),
		uid:     uuid.New().String(),
	}
}

func (app *App) AddClient(uid string, cl *ClientHandler) {
	app.mx.Lock()
	defer app.mx.Unlock()

	app.Logger.Infof("new client: %s", uid)
	app.clients[uid] = cl
}

func (app *App) RemoveClient(uid string) {
	app.mx.Lock()
	defer app.mx.Unlock()

	if _, ok := app.clients[uid]; !ok {
		app.Logger.Infof("remove client: %s", uid)
		delete(app.clients, uid)
	}
}

func (app *App) Run() {
	var cancel context.CancelFunc

	app.ctx, cancel = context.WithCancel(context.Background())

	go func() {
		if err := app.ListenTCP(fmt.Sprintf(":%d", app.port)); err != nil {
			panic(err)
		}
	}()

	go app.EventProcessor()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-c
	app.Logger.Info("exiting...")
	cancel()
}

func (app *App) EventProcessor() {
	for {
		evt := <-app.ch

		switch {
		case evt.Type == "t-x-c-t":
			// ping
			app.Logger.Debugf("ping from %s", evt.Uid)
			app.SendToAll(evt, evt.Uid)
			app.SendMsgTo([]byte(cot.MakePing(app.uid)), evt.Uid)
		case evt.IsChat():
			app.Logger.Infof("chat %s %s", evt.Detail.Chat, evt.GetText())
			app.SendToAll(evt, evt.Uid)
		case strings.HasPrefix(evt.Type, "a-"), strings.HasPrefix(evt.Type, "b-"):
			if evt.GetCallsignTo() != "" {
				app.Logger.Debugf("point or pos %s (%s) for %s", evt.Uid, evt.GetCallsign(), evt.GetCallsignTo())
				app.SendToCallsign(evt, evt.GetCallsignTo())
			} else {
				app.Logger.Debugf("point or pos %s (%s)", evt.Uid, evt.Detail.Contact.Callsign)
				app.SendToAll(evt, evt.Uid)
			}

		default:
			app.Logger.Infof("event: %s %s", evt.Type, evt.Uid)
			app.SendToAll(evt, evt.Uid)
		}
	}
}

func (app *App) SendToAll(evt *cot.Event, author string) {
	app.mx.RLock()
	defer app.mx.RUnlock()

	msg, err := xml.Marshal(evt)

	if err != nil {
		app.Logger.Errorf("marshalling error: %v", err)
		return
	}

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
	app.mx.RLock()
	defer app.mx.RUnlock()

	if h, ok := app.clients[uid]; ok {
		h.AddMsg(msg)
	}
}

func (app *App) SendMsgToCallsign(msg []byte, callsign string) {
	app.mx.RLock()
	defer app.mx.RUnlock()

	for _, h := range app.clients {
		if h.Callsign == callsign {
			h.AddMsg(msg)
		}
	}
}

func main() {
	fmt.Printf("version %s:%s\n", gitBranch, gitRevision)

	var tcpPort = flag.Int("tcp", 8089, "port for udp")

	flag.Parse()

	cfg := zap.NewDevelopmentConfig()
	logger, _ := cfg.Build()
	defer logger.Sync()

	app := NewApp(*tcpPort, logger.Sugar())
	app.Run()
}
