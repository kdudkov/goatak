package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"gotac/cot"
	"gotac/xml"
)

var (
	gitRevision = "unknown"
	gitBranch   = "unknown"
)

type Msg struct {
	event *cot.Event
	dat   []byte
}

type App struct {
	Logger *zap.SugaredLogger
	port   int

	clients map[string]*ClientHandler
	points  map[string]*cot.Point
	mx      sync.RWMutex
	ctx     context.Context
	uid     string
	ch      chan *Msg
	logging bool
}

func NewApp(port int, logger *zap.SugaredLogger) *App {
	return &App{
		Logger:  logger,
		port:    port,
		mx:      sync.RWMutex{},
		ch:      make(chan *Msg, 20),
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

	if _, ok := app.clients[uid]; ok {
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
		msg := <-app.ch

		if msg.event == nil {
			continue
		}

		if app.logging {
			if f, err := os.OpenFile(msg.event.Type+".xml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); err == nil {
				f.Write(msg.dat)
				f.Write([]byte{13})
				f.Close()
			} else {
				fmt.Println(err)
			}
		}

		switch {
		case msg.event.Type == "t-x-c-t":
			// ping
			app.Logger.Debugf("ping from %s", msg.event.Uid)
		case msg.event.IsChat():
			app.Logger.Infof("chat %s %s", msg.event.Detail.Chat, msg.event.GetText())
		case strings.HasPrefix(msg.event.Type, "a-"):
			app.Logger.Debugf("point %s (%s)", msg.event.Uid, msg.event.Detail.Contact.Callsign)
		case strings.HasPrefix(msg.event.Type, "b-"):
			app.Logger.Debugf("pos %s (%s)", msg.event.Uid, msg.event.Detail.Contact.Callsign)
		default:
			app.Logger.Debugf("event: %s", msg.event)
		}

		if len(msg.event.GetCallsignTo()) > 0 {
			for _, s := range msg.event.GetCallsignTo() {
				app.SendMsgToCallsign(msg.dat, s)
			}
		} else {
			app.SendMsgToAll(msg.dat, msg.event.Uid)
		}
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
	app.mx.RLock()
	defer app.mx.RUnlock()

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
	var logging = flag.Bool("logging", false, "port for udp")

	flag.Parse()

	cfg := zap.NewDevelopmentConfig()
	logger, _ := cfg.Build()
	defer logger.Sync()

	app := NewApp(*tcpPort, logger.Sugar())
	app.logging = *logging
	app.Run()
}
