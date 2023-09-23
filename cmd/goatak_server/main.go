package main

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/internal/repository"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/kdudkov/goatak/pkg/model"
)

var (
	gitRevision            = "unknown"
	gitBranch              = "unknown"
	lastSeenOfflineTimeout = time.Minute * 5
)

type EventProcessor func(msg *cot.CotMessage)

type AppConfig struct {
	udpAddr   string
	tcpAddr   string
	adminAddr string
	apiAddr   string
	certAddr  string
	tlsAddr   string

	usersFile string

	logging    bool
	tlsCert    *tls.Certificate
	certPool   *x509.CertPool
	serverCert *x509.Certificate
	ca         []*x509.Certificate

	useSsl bool

	webtakRoot string

	debug bool

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
	feeds    sync.Map

	users repository.UserRepository

	ctx             context.Context
	uid             string
	ch              chan *cot.CotMessage
	eventProcessors map[string]EventProcessor
}

func NewApp(config *AppConfig, logger *zap.SugaredLogger) *App {
	app := &App{
		Logger:          logger,
		config:          config,
		packageManager:  NewPackageManager(logger.Named("packageManager")),
		users:           repository.NewFileUserRepo(logger.Named("userManager"), config.usersFile),
		ch:              make(chan *cot.CotMessage, 20),
		handlers:        sync.Map{},
		items:           repository.NewItemsMemoryRepo(),
		feeds:           sync.Map{},
		uid:             uuid.New().String(),
		eventProcessors: make(map[string]EventProcessor),
	}

	return app
}

func (app *App) Run() {
	app.InitMessageProcessors()
	app.loadFeeds()

	if app.users != nil {
		_ = app.users.Start()
	}

	if err := app.packageManager.Init(); err != nil {
		log.Fatal(err)
	}

	var cancel context.CancelFunc

	app.ctx, cancel = context.WithCancel(context.Background())

	if app.config.udpAddr != "" {
		go func() {
			if err := app.ListenUDP(app.config.udpAddr); err != nil {
				panic(err)
			}
		}()
	}

	if app.config.tcpAddr != "" {
		go func() {
			if err := app.ListenTCP(app.config.tcpAddr); err != nil {
				panic(err)
			}
		}()
	}

	if app.config.tlsCert != nil && app.config.tlsAddr != "" {
		go func() {
			if err := app.listenTls(app.config.tlsAddr); err != nil {
				panic(err)
			}
		}()
	}

	NewHttp(app).Start()

	go app.MessageProcessor()
	go app.cleaner()

	for _, c := range app.config.connections {
		app.Logger.Infof("start external connection to %s", c)
		go app.ConnectTo(c)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-c
	app.Logger.Info("exiting...")
	cancel()
	app.Logger.Info("save feeds")
	app.saveFeeds()
}

func (app *App) NewCotMessage(msg *cot.CotMessage) {
	app.ch <- msg
}

func (app *App) AddClientHandler(ch client.ClientHandler) {
	app.handlers.Store(ch.GetName(), ch)
}

func (app *App) RemoveClientHandler(name string) {
	if _, ok := app.handlers.Load(name); ok {
		app.Logger.Infof("remove handler: %s", name)
		app.handlers.Delete(name)
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
			Scope:      cl.GetScope(),
			TakMessage: cot.MakeOfflineMsg(uid, ""),
		}
		app.SendToAllOther(msg)
	}

	app.RemoveClientHandler(cl.GetName())
}

func (app *App) ConnectTo(addr string) {
	name := "ext_" + addr
	for app.ctx.Err() == nil {
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
			Uid:      app.uid,
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
		break
	case "ssl":
		tlsConn = true
		break
	default:
		return nil, fmt.Errorf("invalid connect string: %s", connectStr)
	}

	addr := fmt.Sprintf("%s:%s", parts[0], parts[1])
	if tlsConn {
		app.Logger.Infof("connecting with SSL to %s...", connectStr)
		conn, err := tls.Dial("tcp", addr, app.getTlsConfig())
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
	} else {
		app.Logger.Infof("connecting to %s...", connectStr)
		return net.DialTimeout("tcp", addr, time.Second*3)
	}
}

func (app *App) getTlsConfig() *tls.Config {
	p12Data, err := ioutil.ReadFile(viper.GetString("ssl.cert"))
	if err != nil {
		app.Logger.Fatal(err)
	}

	key, cert, _, err := pkcs12.DecodeChain(p12Data, viper.GetString("ssl.password"))
	if err != nil {
		app.Logger.Fatal(err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key.(crypto.PrivateKey),
		Leaf:        cert,
	}

	return &tls.Config{Certificates: []tls.Certificate{tlsCert}, InsecureSkipVerify: true}
}

func (app *App) GetCallsign(uid string) string {
	i := app.items.Get(uid)
	if i != nil {
		return i.GetCallsign()
	}
	return ""
}

func (app *App) MessageProcessor() {
	for msg := range app.ch {
		if msg.TakMessage.CotEvent == nil {
			continue
		}

		if c := app.items.Get(msg.GetUid()); c != nil {
			c.Update(nil)
		}

		_, processor := app.GetProcessor(msg.GetType())

		if processor != nil {
			processor(msg)
		} else {
			app.Logger.Warn("unknown message %s", msg.GetType())
			if app.config.logging {
				if err := logToFile(msg); err != nil {
					app.Logger.Errorf("%v", err)
				}
			}
		}

		app.route(msg)
	}
}

func (app *App) route(msg *cot.CotMessage) {
	if len(msg.Detail.GetDest()) > 0 {
		for _, s := range msg.Detail.GetDest() {
			app.SendToCallsign(s, msg.TakMessage)
		}
	} else {
		app.SendToAllOther(msg)
	}
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
			} else {
				if item.IsOnline() && item.GetLastSeen().Add(lastSeenOfflineTimeout).Before(time.Now()) {
					item.SetOffline()
				}
			}
		}
		return true
	})

	for _, uid := range toDelete {
		app.items.Remove(uid)
	}
}

func (app *App) SendToAllOther(msg *cot.CotMessage) {
	app.ForAllClients(func(ch client.ClientHandler) bool {
		if ch.GetName() != msg.From && ch.CanSeeScope(msg.Scope) {
			if err := ch.SendMsg(msg.TakMessage); err != nil {
				app.Logger.Errorf("error sending to %s: %v", ch.GetName(), err)
			}
		}
		return true
	})
}

func (app *App) SendToCallsign(callsign string, msg *cotproto.TakMessage) {
	app.ForAllClients(func(ch client.ClientHandler) bool {
		for _, c := range ch.GetUids() {
			if c == callsign {
				if err := ch.SendMsg(msg); err != nil {
					app.Logger.Errorf("error: %v", err)
				}
				return false
			}
		}
		return true
	})
}

func (app *App) logMessage(c *model.ChatMessage) {
	fd, err := os.OpenFile("msg.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		app.Logger.Errorf("can't write to message log: %s", err.Error())
		return
	}
	defer fd.Close()
	fmt.Fprintf(fd, "%s %s (%s) -> %s (%s) \"%s\"\n", c.Time, c.From, c.FromUid, c.Chatroom, c.ToUid, c.Text)
}

func (app *App) loadFeeds() {
	f, err := os.Open("feeds.yml")
	if err != nil {
		return
	}
	defer f.Close()

	r := make([]*Feed2, 0)
	if err := yaml.NewDecoder(f).Decode(&r); err != nil {
		app.Logger.Errorf("error reading feeds: %s", err.Error())
		return
	}
	for _, feed := range r {
		app.feeds.Store(feed.Uid, feed.ToFeed())
	}
}

func (app *App) saveFeeds() error {
	r := make([]*Feed2, 0)
	app.feeds.Range(func(_, value any) bool {
		if f, ok := value.(*Feed); ok {
			r = append(r, f.ToFeed2())
		}

		return true
	})

	f, err := os.Create("feeds.yml")
	if err != nil {
		return err
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	return enc.Encode(r)
}

func loadPem(name string) ([]*x509.Certificate, error) {
	if name == "" {
		return nil, nil
	}
	pemBytes, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("error loading %s: %s", name, err.Error())
	}
	var certs []*x509.Certificate
	var pemBlock *pem.Block
	for {
		pemBlock, pemBytes = pem.Decode(pemBytes)
		if pemBlock == nil {
			break
		}
		if pemBlock.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(pemBlock.Bytes)
			if err != nil {
				return nil, err
			}
			certs = append(certs, cert)
		}
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no cert in file")
	}
	return certs, nil
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

func main() {
	fmt.Printf("version %s %s\n", gitRevision, gitBranch)
	var debug = flag.Bool("debug", false, "debug node")
	var conf = flag.String("config", "goatak_server.yml", "name of config file")
	flag.Parse()

	viper.SetConfigFile(*conf)

	viper.SetDefault("udp_addr", ":8999")
	viper.SetDefault("tcp_addr", ":8999")
	viper.SetDefault("ssl_addr", ":8089")
	viper.SetDefault("admin_addr", ":8080")
	viper.SetDefault("api_addr", ":8889")
	viper.SetDefault("log", false)

	viper.SetDefault("me.lat", 59.8396)
	viper.SetDefault("me.lon", 31.0213)
	viper.SetDefault("users_file", "users.yml")

	viper.SetDefault("me.zoom", 10)

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
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
		debug:       *debug,
		connections: viper.GetStringSlice("connections"),
		usersFile:   viper.GetString("users_file"),
		webtakRoot:  viper.GetString("webtak_root"),
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
