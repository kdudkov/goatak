package main

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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

	dataDir string

	logging    bool
	tlsCert    *tls.Certificate
	certPool   *x509.CertPool
	serverCert *x509.Certificate
	ca         []*x509.Certificate

	useSsl bool

	webtakRoot string

	debug bool

	certTtlDays int
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
		packageManager:  NewPackageManager(logger.Named("packageManager"), filepath.Join(config.dataDir, "mp")),
		users:           repository.NewFileUserRepo(logger.Named("userManager"), config.usersFile),
		ch:              make(chan *cot.CotMessage, 20),
		handlers:        sync.Map{},
		items:           repository.NewItemsMemoryRepo(),
		feeds:           repository.NewFeedsFileRepo(logger.Named("feedsRepo"), filepath.Join(config.dataDir, "feeds")),
		uid:             uuid.New().String(),
		eventProcessors: make(map[string]EventProcessor),
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
}

func (app *App) NewCotMessage(msg *cot.CotMessage) {
	app.ch <- msg
}

func (app *App) AddEventProcessor(e EventProcessor, mask ...string) {
	for _, s := range mask {
		app.eventProcessors[s] = e
	}
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

func (app *App) MessageProcessor() {
	for msg := range app.ch {
		if app.config.logging {
			if err := app.logToFile(msg); err != nil {
				app.Logger.Warnf("error logging message: %s", err.Error())
			}
		}
		if msg.TakMessage.CotEvent == nil {
			continue
		}

		if c := app.items.Get(msg.GetUid()); c != nil {
			c.Update(nil)
		}

		if _, processor := app.GetProcessor(msg.GetType()); processor != nil {
			processor(msg)
		}

		if !strings.HasPrefix(msg.GetType(), "a-") {
			name, exact := cot.GetMsgType(msg.GetType())
			if exact {
				app.Logger.Debugf("%s %s", msg.GetType(), name)
			} else {
				app.Logger.Infof("%s %s (extended)", msg.GetType(), name)
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

func (app *App) logToFile(msg *cot.CotMessage) error {
	path := filepath.Join(app.config.dataDir, "log")

	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}

	// don't save pings
	if msg.GetType() == "t-x-c-t" || msg.GetType() == "t-x-c-t-r" {
		return nil
	}

	fname := filepath.Join(path, time.Now().Format("2006-01-02.tak"))

	f, err := os.OpenFile(fname, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	d, err := proto.Marshal(msg.TakMessage)
	if err != nil {
		return err
	}
	l := uint32(len(d))
	_, _ = f.Write([]byte{byte(l % 256), byte(l / 256)})
	_, _ = f.Write(d)
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
	viper.SetDefault("data_dir", "data")

	viper.SetDefault("me.lat", 59.8396)
	viper.SetDefault("me.lon", 31.0213)
	viper.SetDefault("users_file", "users.yml")

	viper.SetDefault("me.zoom", 10)
	viper.SetDefault("ssl.cert_ttl_days", 365)

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
		dataDir:     viper.GetString("data_dir"),
		debug:       *debug,
		connections: viper.GetStringSlice("connections"),
		usersFile:   viper.GetString("users_file"),
		webtakRoot:  viper.GetString("webtak_root"),
		certTtlDays: viper.GetInt("ssl.cert_ttl_days"),
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
