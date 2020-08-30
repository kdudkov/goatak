package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"goatac/cot"
	"goatac/model"
	"goatac/xml"
)

const (
	pingTimeout = time.Second * 15
)

type App struct {
	conn      net.Conn
	Logger    *zap.SugaredLogger
	callsign  string
	addr      string
	uid       string
	ch        chan []byte
	lastWrite time.Time
	pingTimer *time.Timer
	unitsMx   sync.RWMutex
	lat       float64
	lon       float64
	units     map[string]*model.Unit
}

func main() {
	var call = flag.String("name", "miner", "callsign")
	//var addr = flag.String("addr", "127.0.0.1:8089", "host:port to connect")
	var addr = flag.String("addr", "discordtakserver.mooo.com:48088", "host:port to connect")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := zap.NewDevelopmentConfig()
	logger, _ := cfg.Build()
	defer logger.Sync()

	app := NewApp(*call, *addr, logger.Sugar())
	go app.Run(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-c
	cancel()
}

func NewApp(callsign string, addr string, logger *zap.SugaredLogger) *App {
	return &App{
		Logger:   logger,
		callsign: callsign,
		addr:     addr,
		lat:      35.462939,
		lon:      -97.537283,
		uid:      makeUid(callsign),
		unitsMx:  sync.RWMutex{},
		units:    make(map[string]*model.Unit, 0),
	}
}

func (app *App) Run(ctx context.Context) {
	go func() {
		if err := NewHttp(app, ":8080").Serve(); err != nil {
			panic(err)
		}
	}()

	for ctx.Err() == nil {
		fmt.Println("connecting...")
		if err := app.connect(); err != nil {
			time.Sleep(time.Second * 5)
			continue
		}

		app.ch = make(chan []byte, 20)
		app.AddEvent(app.MakeMe())

		wg := &sync.WaitGroup{}
		wg.Add(2)
		go app.reader(ctx, wg)
		go app.writer(ctx, wg)
		wg.Wait()

		fmt.Println("disconnected")
	}
}

func makeUid(callsign string) string {
	h := md5.New()
	h.Write([]byte(callsign))
	uid := fmt.Sprintf("%x", h.Sum(nil))
	uid = uid[len(uid)-14:]

	return "ANDROID-" + uid
}

func (app *App) connect() error {
	var err error
	if app.conn, err = net.Dial("tcp", app.addr); err != nil {
		return err
	}

	return nil
}

func (app *App) reader(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	n := 0
	er := cot.NewEventnReader(app.conn)

	for ctx.Err() == nil {
		app.conn.SetReadDeadline(time.Now().Add(time.Second * 120))
		dat, err := er.ReadEvent()
		if err != nil {
			app.Logger.Errorf("read error: %v", err)
			break
		}

		evt := &cot.Event{}
		if err := xml.Unmarshal(dat, evt); err != nil {
			app.Logger.Errorf("decode err: %v", err)
			break
		}
		app.ProcessEvent(evt)
		n++
	}
	app.conn.Close()
	close(app.ch)

	app.Logger.Infof("got %d messages", n)
}

func (app *App) writer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

Loop:
	for {
		select {
		case msg := <-app.ch:
			app.setWriteActivity()
			//if _, err := h.conn.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")); err != nil {
			//	h.stop()
			//	return err
			//}
			if len(msg) == 0 {
				break
			}
			if _, err := app.conn.Write(msg); err != nil {
				break Loop
			}
		case <-ctx.Done():
			break Loop
		}
	}

	app.conn.Close()
}

func (app *App) setWriteActivity() {
	app.lastWrite = time.Now()

	if app.pingTimer == nil {
		app.pingTimer = time.AfterFunc(pingTimeout, app.sendPing)
	} else {
		app.pingTimer.Reset(pingTimeout)
	}
}

func (app *App) AddEvent(evt *cot.Event) bool {
	msg, err := xml.Marshal(evt)
	if err != nil {
		app.Logger.Errorf("marshal error: %v", err)
		return false
	}

	select {
	case app.ch <- msg:
		return true
	default:
		return false
	}

	return false
}

func (app *App) sendPing() {
	if time.Now().Sub(app.lastWrite) > pingTimeout {
		app.Logger.Debug("sending ping")
		app.AddEvent(cot.MakePing(app.uid))
	}
}

func (app *App) ProcessEvent(evt *cot.Event) {
	switch {
	case evt.Type == "t-x-c-t":
		app.Logger.Debugf("ping from %s", evt.Uid)
	case evt.Type == "t-x-c-t-r":
		app.Logger.Debugf("pong")
	case evt.IsChat():
		app.Logger.Infof("message from %s chat %s: %s", evt.Detail.Chat.Sender, evt.Detail.Chat.Room, evt.GetText())
	case strings.HasPrefix(evt.Type, "a-"):
		app.Logger.Debugf("pos %s (%s) %s", evt.Uid, evt.Detail.Contact.Callsign, evt.Type)
		if evt.Stale.After(time.Now()) {
			app.AddUnit(evt.Uid, model.FromEvent(evt))
		}
	case strings.HasPrefix(evt.Type, "b-"):
		app.Logger.Debugf("point %s (%s) %s", evt.Uid, evt.Detail.Contact.Callsign, evt.Type)
		if evt.Stale.After(time.Now()) {
			app.AddUnit(evt.Uid, model.FromEvent(evt))
		}
	default:
		app.Logger.Debugf("event: %s", evt)
	}
}

func (app *App) AddUnit(uid string, u *model.Unit) {
	app.unitsMx.Lock()
	defer app.unitsMx.Unlock()

	app.units[uid] = u
}

func (app *App) MakeMe() *cot.Event {
	ev := cot.BasicEvent("a-f-G-U-C", app.uid, time.Hour)
	ev.Detail = *cot.BasicDetail(app.callsign, "Red", "HQ")
	ev.Point.Lat = app.lat
	ev.Point.Lon = app.lon

	return ev
}
