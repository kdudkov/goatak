package main

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/kdudkov/goatak/cotproto"
	"github.com/kdudkov/goatak/cotxml"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/model"
)

const (
	pingTimeout = time.Second * 15
	alfaNum     = "abcdefghijklmnopqrstuvwxyz012346789"
)

var (
	gitRevision = "unknown"
	gitBranch   = "unknown"
)

type App struct {
	conn      net.Conn
	addr      string
	ver       uint32
	webPort   int
	Logger    *zap.SugaredLogger
	ch        chan []byte
	lastWrite time.Time
	pingTimer *time.Timer
	units     sync.Map
	points    sync.Map

	callsign string
	uid      string
	typ      string
	team     string
	role     string
	lat      float64
	lon      float64
	zoom     int8
	online   uint32
}

func NewApp(uid string, callsign string, addr string, webPort int, logger *zap.SugaredLogger) *App {
	return &App{
		Logger:   logger,
		callsign: callsign,
		uid:      uid,
		addr:     addr,
		webPort:  webPort,
		units:    sync.Map{},
		points:   sync.Map{},
	}
}

func (app *App) setOnline(s bool) {
	if s {
		atomic.StoreUint32(&app.online, 1)
	} else {
		atomic.StoreUint32(&app.online, 0)
	}
}

func (app *App) isOnline() bool {
	return atomic.LoadUint32(&app.online) == 1
}

func (app *App) Run(ctx context.Context) {
	go func() {
		addr := fmt.Sprintf(":%d", app.webPort)
		app.Logger.Infof("listening %s", addr)
		if err := NewHttp(app, addr).Serve(); err != nil {
			panic(err)
		}
	}()

	app.ch = make(chan []byte, 20)

	for ctx.Err() == nil {
		app.Logger.Infof("connecting to %s...", app.addr)
		if err := app.connect(); err != nil {
			time.Sleep(time.Second * 5)
			continue
		}

		app.setOnline(true)

		wg := &sync.WaitGroup{}
		wg.Add(3)

		ctx1, cancel := context.WithCancel(ctx)
		go app.reader(ctx1, wg, cancel)
		go app.writer(ctx1, wg)
		go app.sender(ctx1, wg)
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

func (app *App) reader(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()
	n := 0
	er := cot.NewTagReader(app.conn)
	pr := cot.NewProtoReader(app.conn)
	app.Logger.Infof("start reader")

Loop:
	for ctx.Err() == nil {
		app.conn.SetReadDeadline(time.Now().Add(time.Second * 120))

		var msg *cotproto.TakMessage
		var err error

		switch atomic.LoadUint32(&app.ver) {
		case 0:
			msg, err = app.processXMLRead(er)
		case 1:
			msg, err = app.processProtoRead(pr)
		}

		if err != nil {
			if err == io.EOF {
				break Loop
			}
			app.Logger.Errorf("%v", err)
			break Loop
		}

		if msg == nil {
			continue
		}

		d, err := cotxml.XMLDetailFromString(msg.GetCotEvent().GetDetail().GetXmlDetail())

		if err != nil {
			app.Logger.Errorf("error decoding details: %v", err)
			return
		}

		app.ProcessEvent(&cot.Msg{
			TakMessage: msg,
			Detail:     d,
		})
		n++
	}

	app.setOnline(false)
	app.conn.Close()
	cancel()
	app.Logger.Infof("got %d messages", n)
}

func (app *App) processXMLRead(er *cot.TagReader) (*cotproto.TakMessage, error) {
	tag, dat, err := er.ReadTag()
	if err != nil {
		return nil, err
	}

	if tag == "?xml" {
		return nil, nil
	}

	if tag != "event" {
		return nil, fmt.Errorf("bad tag: %s", dat)
	}

	ev := &cotxml.Event{}
	if err := xml.Unmarshal(dat, ev); err != nil {
		return nil, fmt.Errorf("xml decode error: %v, data: %s", err, string(dat))
	}

	if ev.Detail.TakControl != nil && ev.Detail.TakControl.TakProtocolSupport != nil {
		v := ev.Detail.TakControl.TakProtocolSupport.Version
		app.Logger.Infof("server supports protocol v. %d", v)
		if v >= 1 {
			app.AddEvent(cotxml.VersionReqMsg(1))
		}
		return nil, nil
	}

	if ev.Detail.TakControl != nil && ev.Detail.TakControl.TakResponce != nil {
		ok := ev.Detail.TakControl.TakResponce.Status
		app.Logger.Infof("server switches to v1: %v", ok)
		if ok {
			atomic.StoreUint32(&app.ver, 1)
		}
		return nil, nil
	}

	msg, _ := cot.EventToProto(ev)
	return msg, nil
}

func (app *App) processProtoRead(r *cot.ProtoReader) (*cotproto.TakMessage, error) {
	buf, err := r.ReadProtoBuf()
	if err != nil {
		return nil, err
	}

	msg := new(cotproto.TakMessage)
	if err := proto.Unmarshal(buf, msg); err != nil {

		return nil, fmt.Errorf("failed to decode protobuf: %v", err)
	}

	if msg.GetCotEvent().GetDetail().GetXmlDetail() != "" {
		app.Logger.Debugf("%s %s", msg.CotEvent.Type, msg.CotEvent.Detail.XmlDetail)
	}

	return msg, nil
}

func (app *App) writer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

Loop:
	for {
		if !app.isOnline() {
			break
		}
		select {
		case msg := <-app.ch:
			app.setWriteActivity()
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

func (app *App) sender(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	app.AddMsg(app.MakeMe())

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-time.Tick(time.Minute):
			app.Logger.Debugf("sending pos")
			app.AddMsg(app.MakeMe())
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

func (app *App) AddEvent(evt *cotxml.Event) bool {
	if !app.isOnline() {
		return false
	}
	if atomic.LoadUint32(&app.ver) != 0 {
		return false
	}
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
}

func (app *App) AddMsg(msg *cotproto.TakMessage) bool {
	if !app.isOnline() {
		return false
	}

	switch atomic.LoadUint32(&app.ver) {
	case 0:
		buf, err := xml.Marshal(cot.ProtoToEvent(msg))
		if err != nil {
			app.Logger.Errorf("marshal error: %v", err)
			return false
		}

		return app.tryAddPacket(buf)
	case 1:
		buf1, err := proto.Marshal(msg)
		if err != nil {
			app.Logger.Errorf("marshal error: %v", err)
			return false
		}

		buf := make([]byte, len(buf1)+5)
		buf[0] = 0xbf
		n := binary.PutUvarint(buf[1:], uint64(len(buf1)))
		copy(buf[n+1:], buf1)

		return app.tryAddPacket(buf[:n+len(buf1)+2])
	}

	return false
}

func (app *App) tryAddPacket(msg []byte) bool {
	select {
	case app.ch <- msg:
		return true
	default:
		return false
	}
}

func (app *App) sendPing() {
	if time.Now().Sub(app.lastWrite) > pingTimeout {
		app.Logger.Debug("sending ping")
		app.AddMsg(cot.MakePing(app.uid))
	}
}

func (app *App) ProcessEvent(msg *cot.Msg) {

	switch {
	case msg.GetType() == "t-x-c-t":
		app.Logger.Debugf("ping from %s", msg.GetUid())
		if c := app.GetContact(msg.GetUid()); c != nil {
			c.SetLastSeenNow(nil)
		}
	case msg.GetType() == "t-x-c-t-r":
		app.Logger.Debugf("pong")
	case msg.GetType() == "t-x-d-d":
		app.removeByLink(msg)
	case msg.IsChat():
		app.Logger.Infof("message from ")
	case strings.HasPrefix(msg.GetType(), "a-"):
		if msg.IsContact() {
			if msg.GetUid() == app.uid {
				app.Logger.Info("my own info")
				break
			}
			if c := app.GetContact(msg.GetUid()); c != nil {
				app.Logger.Infof("update pos %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
				c.SetLastSeenNow(msg.TakMessage)
			} else {
				app.Logger.Infof("new contact %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
				app.AddContact(msg.GetUid(), model.ContactFromEvent(msg.TakMessage))
			}
		} else {
			app.Logger.Infof("new unit %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
			app.AddUnit(msg.GetUid(), model.UnitFromEvent(msg.TakMessage))
		}
	case strings.HasPrefix(msg.GetType(), "b-"):
		app.Logger.Infof("point %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
		app.AddPoint(msg.GetUid(), model.UnitFromEvent(msg.TakMessage))
	default:
		app.Logger.Debugf("unknown event: %s", msg.GetType())
	}
}

func (app *App) AddUnit(uid string, u *model.Unit) {
	if u == nil {
		return
	}
	app.units.Store(uid, u)
}

func (app *App) AddPoint(uid string, u *model.Unit) {
	if u == nil {
		return
	}
	app.points.Store(uid, u)
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
			app.Logger.Errorf("invalid object for uid %s: %v", uid, v)
		}
	}
	return nil
}

func (app *App) removeByLink(msg *cot.Msg) {
	if msg.Detail != nil && len(msg.Detail.Link) > 0 {
		uid := msg.Detail.Link[0].Uid
		app.Logger.Debugf("remove %s by message", uid)
		if c := app.GetContact(uid); c != nil {
			c.SetOffline()
		}
	}
}

func (app *App) MakeMe() *cotproto.TakMessage {
	ev := cot.BasicMsg(app.typ, app.uid, time.Hour)
	ev.CotEvent.Lat = app.lat
	ev.CotEvent.Lon = app.lon
	ev.CotEvent.Detail = &cotproto.Detail{
		Contact: &cotproto.Contact{
			Endpoint: "123",
			Callsign: app.callsign,
		},
		Group: &cotproto.Group{
			Name: app.team,
			Role: app.role,
		},
		Takv: &cotproto.Takv{
			Device:   "",
			Platform: "GoATAK web client",
			Os:       "",
			Version:  fmt.Sprintf("%s", gitRevision),
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

func main() {
	var conf = flag.String("config", "goatak_client.yml", "name of config file")
	flag.Parse()

	viper.SetConfigFile(*conf)

	viper.SetDefault("server_address", "127.0.0.1:8089")
	viper.SetDefault("web_port", 8080)
	viper.SetDefault("me.callsign", RandString(10))
	viper.SetDefault("me.lat", 35.462939)
	viper.SetDefault("me.lon", -97.537283)
	viper.SetDefault("me.zoom", 5)
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

	app.lat = viper.GetFloat64("me.lat")
	app.lon = viper.GetFloat64("me.lon")
	app.zoom = int8(viper.GetInt("me.zoom"))
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
