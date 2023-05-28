package main

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
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
	"github.com/kdudkov/goatak/cot"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/kdudkov/goatak/cotproto"
	"github.com/kdudkov/goatak/model"
)

var (
	gitRevision            = "unknown"
	gitBranch              = "unknown"
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
	units    sync.Map
	messages []*model.ChatMessage
	feeds    sync.Map

	userManager *UserManager

	ctx context.Context
	uid string
	ch  chan *cot.CotMessage
}

func NewApp(config *AppConfig, logger *zap.SugaredLogger) *App {
	app := &App{
		Logger:         logger,
		config:         config,
		packageManager: NewPackageManager(logger.Named("packageManager")),
		userManager:    NewUserManager(logger.Named("userManager"), config.usersFile),
		ch:             make(chan *cot.CotMessage, 20),
		handlers:       sync.Map{},
		units:          sync.Map{},
		feeds:          sync.Map{},
		uid:            uuid.New().String(),
	}

	return app
}

func (app *App) Run() {
	app.loadFeeds()

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

func (app *App) AddClientHandler(ch cot.ClientHandler) {
	app.handlers.Store(ch.GetName(), ch)
}

func (app *App) RemoveClientHandler(name string) {
	if _, ok := app.handlers.Load(name); ok {
		app.Logger.Infof("remove handler: %s", name)
		app.handlers.Delete(name)
	}
}

func (app *App) ForAllClients(f func(ch cot.ClientHandler) bool) {
	app.handlers.Range(func(_, value any) bool {
		h := value.(cot.ClientHandler)
		return f(h)
	})
}

func (app *App) RemoveHandlerCb(cl cot.ClientHandler) {
	for uid := range cl.GetUids() {
		if c := app.GetItem(uid); c != nil {
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

		h := cot.NewConnClientHandler(name, conn, &cot.HandlerConfig{
			Logger:    app.Logger,
			MessageCb: app.NewCotMessage,
			RemoveCb: func(ch cot.ClientHandler) {
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

func (app *App) AddItem(uid string, u *model.Item) {
	if u == nil {
		return
	}
	app.units.Store(uid, u)
}

func (app *App) GetItem(uid string) *model.Item {
	if v, ok := app.units.Load(uid); ok {
		return v.(*model.Item)
	}
	return nil
}

func (app *App) GetCallsign(uid string) string {
	i := app.GetItem(uid)
	if i != nil {
		return i.GetCallsign()
	}
	return ""
}

func (app *App) RemoveItem(uid string) {
	if _, ok := app.units.Load(uid); ok {
		app.units.Delete(uid)
	}
}

func (app *App) ProcessItem(msg *cot.CotMessage) {
	cl := model.GetClass(msg)
	if c := app.GetItem(msg.GetUid()); c != nil {
		app.Logger.Debugf("update %s %s (%s) %s", cl, msg.GetUid(), msg.GetCallsign(), msg.GetType())
		c.Update(msg)
	} else {
		app.Logger.Infof("new %s %s (%s) %s", cl, msg.GetUid(), msg.GetCallsign(), msg.GetType())
		app.units.Store(msg.GetUid(), model.FromMsg(msg))
	}
}

func (app *App) removeByLink(msg *cot.CotMessage) {
	if msg.Detail != nil && msg.Detail.Has("link") {
		uid := msg.Detail.GetFirst("link").GetAttr("uid")
		typ := msg.Detail.GetFirst("link").GetAttr("type")
		if uid == "" {
			app.Logger.Warnf("invalid remove message: %s", msg.Detail)
			return
		}
		if v := app.GetItem(uid); v != nil {
			switch v.GetClass() {
			case model.CONTACT:
				app.Logger.Debugf("remove %s by message", uid)
				v.SetOffline()
				return
			case model.UNIT, model.POINT:
				app.Logger.Debugf("remove unit/point %s type %s by message", uid, typ)
				//app.units.Delete(uid)
				return
			}
		}
	}
}

func (app *App) MessageProcessor() {
	for msg := range app.ch {
		if msg.TakMessage.CotEvent == nil {
			continue
		}

		if app.config.logging {
			if f, err := os.OpenFile(msg.GetType()+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); err == nil {
				f.WriteString(msg.TakMessage.String())
				f.Close()
			} else {
				fmt.Println(err)
			}
		}

		if app.config.debug && msg.TakMessage.GetCotEvent().GetDetail().GetXmlDetail() != "" {
			app.Logger.Debugf("details: %s", msg.TakMessage.GetCotEvent().GetDetail().GetXmlDetail())
		}

		switch {
		case msg.GetType() == "t-x-c-t", msg.GetType() == "t-x-c-t-r":
			// ping, pong
			app.Logger.Debugf("ping from %s", msg.GetUid())
			uid := msg.GetUid()
			if strings.HasSuffix(uid, "-ping") {
				uid = uid[:len(uid)-5]
			}
			if c := app.GetItem(uid); c != nil {
				c.Update(nil)
			}
			break
		case msg.GetType() == "t-x-d-d":
			app.removeByLink(msg)
			break
		case msg.IsChat():
			if c := model.MsgToChat(msg); c != nil {
				if c.From == "" {
					c.From = app.GetCallsign(c.FromUid)
				}
				app.Logger.Infof("Chat %s (%s) -> %s (%s) \"%s\"", c.From, c.FromUid, c.Chatroom, c.ToUid, c.Text)
				app.messages = append(app.messages, c)
				app.logMessage(c)
			}
			break
		case strings.HasPrefix(msg.GetType(), "a-"):
			app.ProcessItem(msg)
		case strings.HasPrefix(msg.GetType(), "b-"):
			app.Logger.Infof("point %s (%s) %s", msg.GetUid(), msg.GetCallsign(), msg.GetType())
			app.ProcessItem(msg)
		default:
			app.Logger.Warnf("msg: %s", msg)
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

	app.units.Range(func(key, value any) bool {
		val := value.(*model.Item)

		switch val.GetClass() {
		case model.UNIT, model.POINT:
			if val.IsOld() {
				toDelete = append(toDelete, key.(string))
				app.Logger.Debugf("removing %s %s", val.GetClass(), key)
			}
		case model.CONTACT:
			if val.IsOld() {
				toDelete = append(toDelete, key.(string))
				app.Logger.Debugf("removing contact %s", key)
			} else {
				if val.IsOnline() && val.GetLastSeen().Add(lastSeenOfflineTimeout).Before(time.Now()) {
					val.SetOffline()
				}
			}
		}
		return true
	})

	for _, uid := range toDelete {
		app.units.Delete(uid)
	}
}

func (app *App) SendToAllOther(msg *cot.CotMessage) {
	app.ForAllClients(func(ch cot.ClientHandler) bool {
		if ch.GetName() != msg.From && ch.CanSeeScope(msg.Scope) {
			if err := ch.SendMsg(msg.TakMessage); err != nil {
				app.Logger.Errorf("error sending to %s: %v", ch.GetName(), err)
			}
		}
		return true
	})
}

func (app *App) SendToCallsign(callsign string, msg *cotproto.TakMessage) {
	app.ForAllClients(func(ch cot.ClientHandler) bool {
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

	r := make([]*Feed, 0)
	if err := yaml.NewDecoder(f).Decode(&r); err != nil {
		app.Logger.Errorf("error reading feeds: %s", err.Error())
		return
	}
	for _, feed := range r {
		app.feeds.Store(feed.Uid, feed)
	}
}

func (app *App) saveFeeds() error {
	r := make([]*Feed, 0)
	app.feeds.Range(func(_, value any) bool {
		if f, ok := value.(*Feed); ok {
			r = append(r, f)
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
	for _, c := range ca {
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
	var logging = flag.Bool("logging", false, "save all events to files")
	var debug = flag.Bool("debug", false, "debug node")
	var conf = flag.String("config", "goatak_server.yml", "name of config file")
	flag.Parse()

	viper.SetConfigFile(*conf)

	viper.SetDefault("udp_addr", ":8999")
	viper.SetDefault("tcp_addr", ":8999")
	viper.SetDefault("ssl_addr", ":8089")
	viper.SetDefault("admin_addr", ":8080")
	viper.SetDefault("api_addr", ":8889")

	viper.SetDefault("me.lat", 59.8396)
	viper.SetDefault("me.lon", 31.0213)
	viper.SetDefault("password_file", "")

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
		logging:     *logging,
		debug:       *debug,
		connections: viper.GetStringSlice("connections"),
		usersFile:   viper.GetString("password_file"),
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
