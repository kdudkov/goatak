package main

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/internal/repository"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
	"github.com/kdudkov/goatak/pkg/tlsutil"
)

const (
	dbName = "db.sqlite"
)

var (
	lastSeenOfflineTimeout = time.Minute * 5
)

type AppConfig struct {
	udpAddr   string
	tcpAddr   string
	adminAddr string
	apiAddr   string
	certAddr  string
	tlsAddr   string

	usersFile string

	dataDir string

	logging    bool
	tlsCert    *tls.Certificate
	certPool   *x509.CertPool
	serverCert *x509.Certificate
	ca         []*x509.Certificate

	useSsl bool

	webtakRoot string

	debug    bool
	dataSync bool

	certTTLDays int
	connections []string
}

type App struct {
	Logger         *zap.SugaredLogger
	packageManager *PackageManager
	config         *AppConfig
	lat            float64
	lon            float64
	zoom           int8

	handlers sync.Map
	items    repository.ItemsRepository
	messages []*model.ChatMessage
	feeds    repository.FeedsRepository
	missions *MissionManager

	users repository.UserRepository

	uid             string
	ch              chan *cot.CotMessage
	eventProcessors []*EventProcessor
}

func NewApp(config *AppConfig, logger *zap.SugaredLogger) *App {
	app := &App{
		Logger:          logger,
		config:          config,
		packageManager:  NewPackageManager(logger.Named("packageManager"), filepath.Join(config.dataDir, "mp")),
		users:           repository.NewFileUserRepo(logger.Named("userManager"), config.usersFile),
		ch:              make(chan *cot.CotMessage, 20),
		handlers:        sync.Map{},
		items:           repository.NewItemsMemoryRepo(),
		feeds:           repository.NewFeedsFileRepo(logger.Named("feedsRepo"), filepath.Join(config.dataDir, "feeds")),
		uid:             uuid.NewString(),
		eventProcessors: make([]*EventProcessor, 0),
	}

	if app.config.dataSync {
		db, err := getDatabase()

		if err != nil {
			panic(err)
		}

		app.missions = NewMissionManager(db)
		if err := app.missions.Migrate(); err != nil {
			panic(err)
		}
	}

	return app
}

func (app *App) Run() {
	app.InitMessageProcessors()

	if app.users != nil {
		if err := app.users.Start(); err != nil {
			log.Fatal(err)
		}
	}

	if app.feeds != nil {
		if err := app.feeds.Start(); err != nil {
			log.Fatal(err)
		}
	}

	if err := app.packageManager.Start(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if app.config.udpAddr != "" {
		go func() {
			if err := app.ListenUDP(ctx, app.config.udpAddr); err != nil {
				panic(err)
			}
		}()
	}

	if app.config.tcpAddr != "" {
		go func() {
			if err := app.ListenTCP(ctx, app.config.tcpAddr); err != nil {
				panic(err)
			}
		}()
	}

	if app.config.tlsCert != nil && app.config.tlsAddr != "" {
		go func() {
			if err := app.listenTLS(ctx, app.config.tlsAddr); err != nil {
				panic(err)
			}
		}()
	}

	NewHttp(app).Start()

	go app.MessageProcessor()
	go app.cleaner()

	for _, c := range app.config.connections {
		app.Logger.Infof("start external connection to %s", c)
		go app.ConnectTo(ctx, c)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	app.Logger.Info("exiting...")
	cancel()
}

func (app *App) NewCotMessage(msg *cot.CotMessage) {
	if msg != nil {
		messagesMetric.With(prometheus.Labels{"scope": msg.Scope}).Inc()
		app.ch <- msg
	}
}

func (app *App) AddClientHandler(ch client.ClientHandler) {
	app.handlers.Store(ch.GetName(), ch)
	connectionsMetric.Inc()
}

func (app *App) RemoveClientHandler(name string) {
	if _, ok := app.handlers.Load(name); ok {
		app.Logger.Infof("remove handler: %s", name)

		if _, ok := app.handlers.LoadAndDelete(name); ok {
			connectionsMetric.Dec()
		}
	}
}

func (app *App) ForAllClients(f func(ch client.ClientHandler) bool) {
	app.handlers.Range(func(_, value any) bool {
		h := value.(client.ClientHandler)

		return f(h)
	})
}

func (app *App) RemoveHandlerCb(cl client.ClientHandler) {
	for uid := range cl.GetUids() {
		if c := app.items.Get(uid); c != nil {
			c.SetOffline()
		}

		msg := &cot.CotMessage{
			From:       cl.GetName(),
			Scope:      cl.GetUser().GetScope(),
			TakMessage: cot.MakeOfflineMsg(uid, ""),
		}
		app.NewCotMessage(msg)
	}

	app.RemoveClientHandler(cl.GetName())
}

func (app *App) NewContactCb(uid, callsign string) {
	app.Logger.Infof("new contact: %s %s", uid, callsign)
}

func (app *App) ConnectTo(ctx context.Context, addr string) {
	name := "ext_" + addr

	for ctx.Err() == nil {
		conn, err := app.connect(addr)
		if err != nil {
			app.Logger.Errorf("connect error: %s", err)
			time.Sleep(time.Second * 5)

			continue
		}

		app.Logger.Info("connected")

		wg := &sync.WaitGroup{}
		wg.Add(1)

		h := client.NewConnClientHandler(name, conn, &client.HandlerConfig{
			Logger:    app.Logger,
			MessageCb: app.NewCotMessage,
			RemoveCb: func(ch client.ClientHandler) {
				wg.Done()
				app.handlers.Delete(name)
				app.Logger.Info("disconnected")
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
		app.Logger.Infof("connecting with SSL to %s...", connectStr)

		conn, err := tls.Dial("tcp", addr, app.getTLSConfig())
		if err != nil {
			return nil, err
		}

		app.Logger.Debugf("handshake...")

		if err := conn.Handshake(); err != nil {
			return conn, err
		}

		cs := conn.ConnectionState()

		app.Logger.Infof("Handshake complete: %t", cs.HandshakeComplete)
		app.Logger.Infof("version: %d", cs.Version)

		for i, cert := range cs.PeerCertificates {
			app.Logger.Infof("cert #%d subject: %s", i, cert.Subject.String())
			app.Logger.Infof("cert #%d issuer: %s", i, cert.Issuer.String())
			app.Logger.Infof("cert #%d dns_names: %s", i, strings.Join(cert.DNSNames, ","))
		}

		return conn, nil
	}

	app.Logger.Infof("connecting to %s...", connectStr)

	return net.DialTimeout("tcp", addr, time.Second*3)
}

func (app *App) getTLSConfig() *tls.Config {
	p12Data, err := os.ReadFile(viper.GetString("ssl.cert"))
	if err != nil {
		app.Logger.Fatal(err)
	}

	key, cert, _, err := pkcs12.DecodeChain(p12Data, viper.GetString("ssl.password"))
	if err != nil {
		app.Logger.Fatal(err)
	}

	tlsCert := tls.Certificate{ //nolint:exhaustruct
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key.(crypto.PrivateKey),
		Leaf:        cert,
	}

	return &tls.Config{Certificates: []tls.Certificate{tlsCert}, InsecureSkipVerify: true}
}

func (app *App) MessageProcessor() {
	for msg := range app.ch {
		for _, prc := range app.eventProcessors {
			if cot.MatchAnyPattern(msg.GetType(), prc.include...) {
				app.Logger.Debugf("msg is processed by %s", prc.name)
				prc.cb(msg)
			}
		}

		app.route(msg)
	}
}

func (app *App) route(msg *cot.CotMessage) {
	if missions := msg.GetDetail().GetDestMission(); len(missions) > 0 {
		for _, name := range missions {
			app.missions.AddPoint(name, msg)

			for _, uid := range app.missions.GetSubscribers(name) {
				app.SendToUID(uid, msg)
			}
		}

		return
	}

	if dest := msg.GetDetail().GetDestCallsign(); len(dest) > 0 {
		for _, s := range dest {
			app.SendToCallsign(s, msg)
		}

		return
	}

	app.SendBroadcast(msg)
}

func (app *App) cleaner() {
	for range time.Tick(time.Minute) {
		app.cleanOldUnits()
	}
}

func (app *App) cleanOldUnits() {
	toDelete := make([]string, 0)

	app.items.ForEach(func(item *model.Item) bool {
		switch item.GetClass() {
		case model.UNIT, model.POINT:
			if item.IsOld() {
				toDelete = append(toDelete, item.GetUID())
				app.Logger.Debugf("removing %s %s", item.GetClass(), item.GetUID())
			}
		case model.CONTACT:
			if item.IsOld() {
				toDelete = append(toDelete, item.GetUID())
				app.Logger.Debugf("removing contact %s", item.GetUID())
			} else if item.IsOnline() && item.GetLastSeen().Add(lastSeenOfflineTimeout).Before(time.Now()) {
				item.SetOffline()
			}
		}

		return true
	})

	for _, uid := range toDelete {
		app.items.Remove(uid)
		app.missions.DeletePoint(uid)
	}
}

func (app *App) SendBroadcast(msg *cot.CotMessage) {
	app.ForAllClients(func(ch client.ClientHandler) bool {
		if ch.GetName() != msg.From {
			if err := ch.SendMsg(msg); err != nil {
				app.Logger.Errorf("error sending to %s: %v", ch.GetName(), err)
			}
		}

		return true
	})
}

func (app *App) SendToCallsign(callsign string, msg *cot.CotMessage) {
	app.ForAllClients(func(ch client.ClientHandler) bool {
		for _, c := range ch.GetUids() {
			if c == callsign {
				if err := ch.SendMsg(msg); err != nil {
					app.Logger.Errorf("error: %v", err)
				}
			}
		}

		return true
	})
}

func (app *App) SendToUID(uid string, msg *cot.CotMessage) {
	app.ForAllClients(func(ch client.ClientHandler) bool {
		if ch.HasUID(uid) {
			if err := ch.SendMsg(msg); err != nil {
				app.Logger.Errorf("error: %v", err)
			}
		}

		return true
	})
}

func loadPem(name string) ([]*x509.Certificate, error) {
	if name == "" {
		return nil, nil
	}

	pemBytes, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("error loading %s: %s", name, err.Error())
	}

	return tlsutil.DecodeAllCerts(pemBytes)
}

func processCerts(conf *AppConfig) error {
	for _, k := range []string{"ssl.ca", "ssl.cert", "ssl.key"} {
		if viper.GetString(k) == "" {
			return nil
		}
	}

	roots := x509.NewCertPool()
	conf.certPool = roots

	ca, err := loadPem(viper.GetString("ssl.ca"))
	if err != nil {
		return err
	}

	for _, c := range ca {
		roots.AddCert(c)
	}

	conf.ca = ca

	cert, err := loadPem(viper.GetString("ssl.cert"))
	if err != nil {
		return err
	}

	if len(cert) > 0 {
		conf.serverCert = cert[0]
	}

	for _, c := range cert {
		roots.AddCert(c)
	}

	tlsCert, err := tls.LoadX509KeyPair(viper.GetString("ssl.cert"), viper.GetString("ssl.key"))
	if err != nil {
		return err
	}

	conf.tlsCert = &tlsCert

	return nil
}

func getDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func main() {
	fmt.Printf("version %s\n", getVersion())

	debug := flag.Bool("debug", false, "debug mode")
	conf := flag.String("config", "goatak_server.yml", "name of config file")
	flag.Parse()

	viper.SetConfigFile(*conf)

	viper.SetDefault("udp_addr", ":8999")
	viper.SetDefault("tcp_addr", ":8999")
	viper.SetDefault("ssl_addr", ":8089")
	viper.SetDefault("api_addr", ":8080")
	viper.SetDefault("log", false)
	viper.SetDefault("data_dir", "data")

	viper.SetDefault("me.lat", 59.8396)
	viper.SetDefault("me.lon", 31.0213)
	viper.SetDefault("users_file", "users.yml")

	viper.SetDefault("me.zoom", 10)
	viper.SetDefault("ssl.cert_ttl_days", 365)

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	flag.Parse()

	var cfg zap.Config
	if *debug {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	logger, _ := cfg.Build()
	defer logger.Sync()

	config := &AppConfig{
		udpAddr:     viper.GetString("udp_addr"),
		tcpAddr:     viper.GetString("tcp_addr"),
		adminAddr:   viper.GetString("admin_addr"),
		apiAddr:     viper.GetString("api_addr"),
		certAddr:    viper.GetString("cert_addr"),
		tlsAddr:     viper.GetString("ssl_addr"),
		useSsl:      viper.GetBool("ssl.use_ssl"),
		logging:     viper.GetBool("log"),
		dataDir:     viper.GetString("data_dir"),
		debug:       *debug,
		connections: viper.GetStringSlice("connections"),
		usersFile:   viper.GetString("users_file"),
		webtakRoot:  viper.GetString("webtak_root"),
		certTTLDays: viper.GetInt("ssl.cert_ttl_days"),
		dataSync:    viper.GetBool("datasync"),
	}

	if err := processCerts(config); err != nil {
		logger.Error(err.Error())
	}

	app := NewApp(config, logger.Sugar())

	app.lat = viper.GetFloat64("me.lat")
	app.lon = viper.GetFloat64("me.lon")
	app.zoom = int8(viper.GetInt("me.zoom"))
	app.Run()
}
