package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/model"
	"github.com/kdudkov/goatak/xml"
)

const (
	pingTimeout = time.Second * 15
	alfaNum     = "abcdefghijklmnopqrstuvwxyz012346789"

	cleanTimeout = time.Minute * 10
)

var (
	gitRevision = "unknown"
	gitBranch   = "unknown"
)

type App struct {
	conn      net.Conn
	addr      string
	webPort   int
	Logger    *zap.SugaredLogger
	ch        chan []byte
	lastWrite time.Time
	pingTimer *time.Timer
	unitsMx   sync.RWMutex
	units     map[string]*model.Unit

	callsign string
	uid      string
	typ      string
	team     string
	role     string
	lat      float64
	lon      float64
}

func NewApp(uid string, callsign string, addr string, webPort int, logger *zap.SugaredLogger) *App {
	return &App{
		Logger:   logger,
		callsign: callsign,
		uid:      uid,
		addr:     addr,
		webPort:  webPort,
		unitsMx:  sync.RWMutex{},
		units:    make(map[string]*model.Unit, 0),
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

	for ctx.Err() == nil {
		app.Logger.Infof("connecting to %s...", app.addr)
		if err := app.connect(); err != nil {
			time.Sleep(time.Second * 5)
			continue
		}

		app.ch = make(chan []byte, 20)
		app.AddEvent(app.MakeMe())

		wg := &sync.WaitGroup{}
		wg.Add(3)

		stopCh := make(chan bool)

		go app.reader(ctx, wg, stopCh)
		go app.writer(ctx, wg, stopCh)
		go app.sender(ctx, wg, stopCh)
		wg.Wait()

		app.Logger.Info("disconnected")
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

func (app *App) reader(ctx context.Context, wg *sync.WaitGroup, ch chan bool) {
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
	close(ch)
	app.Logger.Infof("got %d messages", n)
}

func (app *App) writer(ctx context.Context, wg *sync.WaitGroup, ch chan bool) {
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
		case <-ch:
			break Loop
		}
	}

	app.conn.Close()
}

func (app *App) sender(ctx context.Context, wg *sync.WaitGroup, ch chan bool) {
	defer wg.Done()

Loop:
	for ctx.Err() == nil {
		select {
		case <-ch:
			break Loop
		case <-time.Tick(time.Second * 10):
			app.AddEvent(app.MakeMe())
		}
	}
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
		if evt.Stale.After(time.Now()) {
			app.Logger.Infof("pos %s (%s) %s", evt.Uid, evt.Detail.Contact.Callsign, evt.Type)
			app.AddUnit(evt.Uid, model.FromEvent(evt))
		} else {
			app.Logger.Debugf("pos %s (%s) %s - stale %s", evt.Uid, evt.Detail.Contact.Callsign, evt.Type, evt.Stale)
		}
	case strings.HasPrefix(evt.Type, "b-"):
		app.Logger.Infof("point %s (%s) %s", evt.Uid, evt.Detail.Contact.Callsign, evt.Type)
		if evt.Stale.After(time.Now()) {
			app.AddUnit(evt.Uid, model.FromEvent(evt))
		}
	default:
		app.Logger.Infof("event: %s", evt)
	}
}

func (app *App) AddUnit(uid string, u *model.Unit) {
	if u == nil {
		return
	}

	app.unitsMx.Lock()
	defer app.unitsMx.Unlock()

	app.units[uid] = u
}

func (app *App) MakeMe() *cot.Event {
	ev := cot.BasicEvent(app.typ, app.uid, time.Hour)
	ev.Detail = *cot.BasicDetail(app.callsign, app.team, app.role)
	ev.Point.Lat = app.lat
	ev.Point.Lon = app.lon
	ev.Detail.TakVersion.Platform = "GoATAK web client"
	ev.Detail.TakVersion.Version = fmt.Sprintf("%s:%s", gitBranch, gitRevision)

	return ev
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
	app.unitsMx.Lock()
	defer app.unitsMx.Unlock()

	toDelete := make([]string, 0)
	for k, v := range app.units {
		if v.Stale.Add(cleanTimeout).Before(time.Now()) {
			toDelete = append(toDelete, k)
			app.Logger.Debugf("removing %s (stale %s)", k, v.Stale.Sub(time.Now()))
		}
	}
	for _, uid := range toDelete {
		delete(app.units, uid)
	}
}

func RandString(strlen int) string {
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = alfaNum[rand.Intn(len(alfaNum))]
	}
	return string(result)
}

func main() {
	var conf = flag.String("config", "atak-web.yml", "name of config file")
	flag.Parse()

	viper.SetConfigFile(*conf)

	viper.SetDefault("server_address", "127.0.0.1:8089")
	viper.SetDefault("web_port", 8080)
	viper.SetDefault("me.callsign", RandString(10))
	viper.SetDefault("me.uid", RandString(16))
	viper.SetDefault("me.lat", 35.462939)
	viper.SetDefault("me.lon", -97.537283)
	viper.SetDefault("me.type", "a-f-G-U-C")
	viper.SetDefault("me.team", "Blue")
	viper.SetDefault("me.role", "HQ")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cfg := zap.NewDevelopmentConfig()
	logger, _ := cfg.Build()
	defer logger.Sync()

	uid := viper.GetString("me.uid")
	if uid == "auto" {
		uid = makeUid(viper.GetString("me.callsign"))
	}

	app := NewApp(
		uid,
		viper.GetString("me.callsign"),
		viper.GetString("server_address"),
		viper.GetInt("web_port"),
		logger.Sugar(),
	)

	app.lat = viper.GetFloat64("me.lat")
	app.lon = viper.GetFloat64("me.lon")
	app.typ = viper.GetString("me.type")
	app.team = viper.GetString("me.team")
	app.role = viper.GetString("me.role")

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
