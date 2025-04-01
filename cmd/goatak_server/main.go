package main

import (
	"context"
	"crypto"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/kdudkov/goatak/cmd/goatak_server/database"
	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/internal/pm"
	"github.com/kdudkov/goatak/internal/repository"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
)

type App struct {
	logger *slog.Logger
	files  *pm.BlobManager
	config *AppConfig
	lat    float64
	lon    float64
	zoom   int8

	handlers sync.Map

	items    repository.ItemsRepository
	messages []*model.ChatMessage
	feeds    repository.FeedsRepository
	dbm      *database.DatabaseManager

	users repository.UserRepository

	uid             string
	ch              chan *cot.CotMessage
	eventProcessors []*EventProcessor
}

func NewApp(config *AppConfig) *App {
	app := &App{
		logger:          slog.Default(),
		config:          config,
		files:           pm.NewBlobManages(filepath.Join(config.DataDir(), "blob")),
		users:           repository.NewFileUserRepo(config.UsersFile()),
		ch:              make(chan *cot.CotMessage, 100),
		handlers:        sync.Map{},
		items:           repository.NewItemsMemoryRepo(),
		feeds:           repository.NewFeedsFileRepo(filepath.Join(config.DataDir(), "feeds")),
		uid:             uuid.NewString(),
		eventProcessors: make([]*EventProcessor, 0),
	}

	db, err := database.GetDatabase(config.String("db"), false)

	if err != nil {
		panic(err)
	}

	app.dbm = database.New(db)
	if err := app.dbm.Migrate(); err != nil {
		panic(err)
	}

	return app
}

func (app *App) Run() {
	app.InitMessageProcessors()

	if err := app.items.Start(); err != nil {
		log.Fatal(err)
	}

	if err := app.users.Start(); err != nil {
		log.Fatal(err)
	}

	if app.feeds != nil {
		if err := app.feeds.Start(); err != nil {
			log.Fatal(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	if addr := app.config.k.String("udp_addr"); addr != "" {
		go func() {
			if err := app.ListenUDP(ctx, addr); err != nil {
				panic(err)
			}
		}()
	}

	if addr := app.config.k.String("tcp_addr"); addr != "" {
		go func() {
			if err := app.ListenTCP(ctx, addr); err != nil {
				panic(err)
			}
		}()
	}

	if addr := app.config.k.String("ssl_addr"); addr != "" && app.config.tlsCert != nil {
		go func() {
			if err := app.listenTLS(ctx, addr); err != nil {
				panic(err)
			}
		}()
	}

	NewHttp(app).Start()

	go app.messageProcessLoop()

	for _, c := range app.config.Connections() {
		app.logger.Info("start external connection to " + c)
		go app.ConnectTo(ctx, c)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	app.logger.Info("exiting...")
	cancel()
}

func (app *App) NewCotMessage(msg *cot.CotMessage) {
	if msg != nil {
		t := msg.GetType()

		if strings.HasPrefix(t, "a-") && len(t) > 5 {
			t = t[:5]
		}

		messagesMetric.With(prometheus.Labels{"scope": msg.Scope, "msg_type": t}).Inc()

		select {
		case app.ch <- msg:
		default:
			dropMetric.With(prometheus.Labels{"scope": msg.Scope, "reason": "main_ch"}).Inc()
		}
	}
}

func (app *App) AddClientHandler(ch client.ClientHandler) {
	app.handlers.Store(ch.GetName(), ch)
	connectionsMetric.With(prometheus.Labels{"scope": ch.GetDevice().GetScope()}).Inc()
}

func (app *App) RemoveClientHandler(name string) {
	if v, ok := app.handlers.LoadAndDelete(name); ok {
		app.logger.Info("remove handler: " + name)
		ch := v.(client.ClientHandler)
		connectionsMetric.With(prometheus.Labels{"scope": ch.GetDevice().GetScope()}).Dec()
	}
}

func (app *App) ForAllClients(f func(ch client.ClientHandler) bool) {
	app.handlers.Range(func(_, value any) bool {
		h := value.(client.ClientHandler)

		return f(h)
	})
}

func (app *App) RemoveHandlerCb(cl client.ClientHandler) {
	app.RemoveClientHandler(cl.GetName())

	for uid := range cl.GetUids() {
		if c := app.items.Get(uid); c != nil {
			c.SetOffline()
		}

		msg := &cot.CotMessage{
			From:       cl.GetName(),
			Scope:      cl.GetDevice().GetScope(),
			TakMessage: cot.MakeOfflineMsg(uid, ""),
		}
		app.NewCotMessage(msg)
	}
}

func (app *App) NewContactCb(uid, callsign string) {
	app.logger.Info(fmt.Sprintf("new contact: %s %s", uid, callsign))
}

func (app *App) ConnectTo(ctx context.Context, addr string) {
	name := "ext_" + addr

	for ctx.Err() == nil {
		conn, err := app.connect(addr)
		if err != nil {
			app.logger.Error("connect error", slog.Any("error", err))
			time.Sleep(time.Second * 5)

			continue
		}

		app.logger.Info("connected")

		wg := &sync.WaitGroup{}
		wg.Add(1)

		h := client.NewConnClientHandler(name, conn, &client.HandlerConfig{
			MessageCb: app.NewCotMessage,
			RemoveCb: func(ch client.ClientHandler) {
				wg.Done()
				app.handlers.Delete(name)
				app.logger.Info("disconnected")
			},
			IsClient: true,
			UID:      app.uid,
		})

		go h.Start()
		app.AddClientHandler(h)

		wg.Wait()
	}
}

func (app *App) connect(connectStr string) (net.Conn, error) {
	parts := strings.Split(connectStr, ":")

	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid connect string: %s", connectStr)
	}

	var tlsConn bool

	switch parts[2] {
	case "tcp":
		tlsConn = false
	case "ssl":
		tlsConn = true
	default:
		return nil, fmt.Errorf("invalid connect string: %s", connectStr)
	}

	addr := fmt.Sprintf("%s:%s", parts[0], parts[1])

	if tlsConn {
		app.logger.Info(fmt.Sprintf("connecting with SSL to %s...", connectStr))

		conn, err := tls.Dial("tcp", addr, app.getTLSConfig())
		if err != nil {
			return nil, err
		}

		app.logger.Debug("handshake...")

		if err := conn.Handshake(); err != nil {
			return conn, err
		}

		cs := conn.ConnectionState()

		app.logger.Info(fmt.Sprintf("Handshake complete: %t", cs.HandshakeComplete))
		app.logger.Info(fmt.Sprintf("version: %d", cs.Version))

		for i, cert := range cs.PeerCertificates {
			app.logger.Info(fmt.Sprintf("cert #%d subject: %s", i, cert.Subject.String()))
			app.logger.Info(fmt.Sprintf("cert #%d issuer: %s", i, cert.Issuer.String()))
			app.logger.Info(fmt.Sprintf("cert #%d dns_names: %s", i, strings.Join(cert.DNSNames, ",")))
		}

		return conn, nil
	}

	app.logger.Info(fmt.Sprintf("connecting to %s...", connectStr))

	return net.DialTimeout("tcp", addr, time.Second*3)
}

func (app *App) getTLSConfig() *tls.Config {
	p12Data, err := os.ReadFile(app.config.k.String("ssl.cert"))
	if err != nil {
		app.logger.Error(err.Error())
		panic(err)
	}

	key, cert, _, err := pkcs12.DecodeChain(p12Data, app.config.k.String("ssl.password"))
	if err != nil {
		app.logger.Error(err.Error())
		panic(err)
	}

	tlsCert := tls.Certificate{ //nolint:exhaustruct,typeassert
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key.(crypto.PrivateKey),
		Leaf:        cert,
	}

	return &tls.Config{Certificates: []tls.Certificate{tlsCert}, InsecureSkipVerify: true} //nolint:exhaustruct
}

func (app *App) messageProcessLoop() {
	for msg := range app.ch {
		app.processMessage(msg)
	}
}

func (app *App) route(msg *cot.CotMessage) bool {
	if missions := msg.GetDetail().GetDestMission(); len(missions) > 0 {
		app.logger.Debug(fmt.Sprintf("point %s %s: missions: %s", msg.GetUID(), msg.GetCallsign(), strings.Join(missions, ",")))

		for _, missionName := range missions {
			app.processMissionPoint(missionName, msg)
		}

		return true
	}

	if dest := msg.GetDetail().GetDestCallsign(); len(dest) > 0 {
		for _, s := range dest {
			app.logger.Info(fmt.Sprintf("point %s %s -> callsign %s", msg.GetUID(), msg.GetCallsign(), s))
			app.sendToCallsign(s, msg)
		}

		return true
	}

	app.sendBroadcast(msg)

	return true
}

func (app *App) processMissionPoint(missionName string, msg *cot.CotMessage) {
	m := app.dbm.MissionQuery().Scope(msg.Scope).Name(missionName).Full().One()

	if m == nil {
		return
	}

	var change *model.Change

	if msg.GetType() == "t-x-d-d" {
		if uid := msg.GetFirstLink("p-p").GetAttr("uid"); uid != "" {
			change = app.dbm.DeleteMissionPoint(m, uid, "")
		}
	} else {
		change = app.dbm.AddMissionPoint(m, msg)
	}

	if change != nil {
		app.notifyMissionSubscribers(m, change)
	}
}

func (app *App) notifyMissionSubscribers(mission *model.Mission, c *model.Change) {
	if mission == nil || c == nil {
		return
	}

	msg := model.MissionChangeNotificationMsg(mission.Name, mission.Scope, c)
	for _, uid := range app.dbm.GetSubscribers(mission.ID) {
		app.sendToUID(uid, msg)
	}
}

func (app *App) sendBroadcast(msg *cot.CotMessage) {
	app.ForAllClients(func(ch client.ClientHandler) bool {
		if ch.GetName() != msg.From {
			if err := ch.SendMsg(msg); err != nil {
				app.logger.Error(fmt.Sprintf("error sending to %s: %v", ch.GetName(), err))
			}
		}

		return true
	})
}

func (app *App) sendToCallsign(callsign string, msg *cot.CotMessage) {
	app.ForAllClients(func(ch client.ClientHandler) bool {
		for _, c := range ch.GetUids() {
			if c == callsign {
				if err := ch.SendMsg(msg); err != nil {
					app.logger.Error("send error", slog.Any("error", err))
				}
			}
		}

		return true
	})
}

func (app *App) sendToUID(uid string, msg *cot.CotMessage) {
	app.ForAllClients(func(ch client.ClientHandler) bool {
		if ch.HasUID(uid) {
			if err := ch.SendMsg(msg); err != nil {
				app.logger.Error("send error", slog.Any("error", err))
			}
		}

		return true
	})
}

func (app *App) checkUID(uid string) bool {
	u := strings.ToLower(uid)
	for _, s := range app.config.BlacklistedUID() {
		if u == strings.ToLower(s) {
			return false
		}
	}

	return true
}

func main() {
	fmt.Printf("version %s\n", getVersion())

	configName := flag.String("config", "goatak_server.yml", "name of config file")
	flag.Parse()

	config := NewAppConfig()
	config.Load(*configName)
	_ = config.LoadEnv("GOATAK_")

	var h slog.Handler
	if config.Bool("debug") {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	slog.SetDefault(slog.New(h))

	if err := config.processCerts(); err != nil {
		slog.Default().Error(err.Error())
	}

	app := NewApp(config)

	app.lat = config.Float64("me.lat")
	app.lon = config.Float64("me.lon")
	app.zoom = int8(config.Int("me.zoom"))
	app.Run()
}
