package client

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

const (
	idleTimeout = 5 * time.Minute
	pingTimeout = time.Second * 15
)

type HandlerConfig struct {
	User         *model.User
	Serial       string
	UID          string
	IsClient     bool
	MessageCb    func(msg *cot.CotMessage)
	RemoveCb     func(ch ClientHandler)
	NewContactCb func(uid, callsign string)
	RoutePings   bool
	Logger       *slog.Logger
	DropMetric   *prometheus.CounterVec
}

type ClientHandler interface {
	GetName() string
	HasUID(uid string) bool
	GetUids() map[string]string
	GetUser() *model.User
	GetSerial() string
	GetVersion() int32
	SendMsg(msg *cot.CotMessage) error
	GetLastSeen() *time.Time
	CanSeeScope(scope string) bool
	Stop()
}

type ConnClientHandler struct {
	cancel       context.CancelFunc
	conn         net.Conn
	addr         string
	localUID     string
	ver          int32
	isClient     bool
	routePings   bool
	uids         sync.Map
	lastActivity atomic.Pointer[time.Time]
	closeTimer   *time.Timer
	sendChan     chan []byte
	active       int32
	user         *model.User
	serial       string
	messageCb    func(msg *cot.CotMessage)
	removeCb     func(ch ClientHandler)
	newContactCb func(uid, callsign string)
	logger       *slog.Logger
	dropMetric   *prometheus.CounterVec
}

func NewConnClientHandler(name string, conn net.Conn, config *HandlerConfig) *ConnClientHandler {
	c := &ConnClientHandler{
		addr:         name,
		conn:         conn,
		ver:          0,
		sendChan:     make(chan []byte, 10),
		active:       1,
		uids:         sync.Map{},
		lastActivity: atomic.Pointer[time.Time]{},
	}

	if config != nil {
		c.user = config.User
		c.serial = config.Serial
		c.localUID = config.UID
		c.isClient = config.IsClient
		c.routePings = config.RoutePings
		c.messageCb = config.MessageCb
		c.removeCb = config.RemoveCb
		c.newContactCb = config.NewContactCb
		c.dropMetric = config.DropMetric

		params := []any{"client", name}

		if u := config.User; u != nil {
			params = append(params, "login", u.GetLogin(), "scope", u.GetScope())
		}

		if config.Serial != "" {
			params = append(params, "cert_sn", config.Serial)
		}

		if config.Logger != nil {
			c.logger = config.Logger.With(params...)
		} else {
			c.logger = slog.Default().With(params...)
		}
	}

	c.closeTimer = time.AfterFunc(idleTimeout, c.closeIdle)

	return c
}

func (h *ConnClientHandler) GetName() string {
	return h.addr
}

func (h *ConnClientHandler) CanSeeScope(scope string) bool {
	return h.user.CanSeeScope(scope)
}

func (h *ConnClientHandler) GetUser() *model.User {
	return h.user
}

func (h *ConnClientHandler) GetSerial() string {
	return h.serial
}

func (h *ConnClientHandler) GetUids() map[string]string {
	res := make(map[string]string)

	h.uids.Range(func(key, value any) bool {
		res[key.(string)] = value.(string)

		return true
	})

	return res
}

func (h *ConnClientHandler) HasUID(uid string) bool {
	_, ok := h.uids.Load(uid)

	return ok
}

func (h *ConnClientHandler) IsActive() bool {
	return atomic.LoadInt32(&h.active) == 1
}

func (h *ConnClientHandler) GetLastSeen() *time.Time {
	return h.lastActivity.Load()
}

func (h *ConnClientHandler) Start() {
	h.logger.Info("starting")

	var ctx context.Context
	ctx, h.cancel = context.WithCancel(context.Background())

	go h.handleWrite()
	go h.handleRead(ctx)

	if h.isClient {
		go h.pinger(ctx)
	}

	if !h.isClient {
		h.logger.Debug("send version msg")

		if err := h.sendEvent(cot.VersionSupportMsg(1)); err != nil {
			h.logger.Error("error sending ver req", "error", err.Error())
		}
	}
}

func (h *ConnClientHandler) pinger(ctx context.Context) {
	ticker := time.NewTicker(pingTimeout)
	defer ticker.Stop()

	for ctx.Err() == nil {
		select {
		case <-ticker.C:
			h.logger.Debug("ping")

			if err := h.SendCot(cot.MakePing(h.localUID)); err != nil {
				h.logger.Debug("sendMsg error", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *ConnClientHandler) handleRead(ctx context.Context) {
	defer h.Stop()

	er := cot.NewTagReader(h.conn)
	pr := cot.NewProtoReader(h.conn)

	for ctx.Err() == nil {
		var msg *cot.CotMessage

		var err error

		switch h.GetVersion() {
		case 0:
			msg, err = h.processXMLRead(er)
		case 1:
			msg, err = h.processProtoRead(pr)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				h.logger.Info("EOF")

				break
			}

			h.logger.Warn("error", "error", err.Error())

			break
		}

		if msg == nil {
			continue
		}

		msg.From = h.addr
		msg.Scope = h.GetUser().GetScope()

		// add new contact uid
		if msg.IsContact() {
			uid := msg.GetUID()
			uid = strings.TrimSuffix(uid, "-ping")

			if _, present := h.uids.Swap(uid, msg.GetCallsign()); !present {
				if h.newContactCb != nil {
					h.newContactCb(uid, msg.GetCallsign())
				}
			}
		}

		// remove contact
		if msg.GetType() == "t-x-d-d" && msg.GetDetail().Has("link") {
			uid := msg.GetDetail().GetFirst("link").GetAttr("uid")
			h.logger.Debug(fmt.Sprintf("delete uid %s by message", uid))
			h.uids.Delete(uid)
		}

		// ping
		if msg.GetType() == "t-x-c-t" {
			h.logger.Debug(fmt.Sprintf("ping from %s %s", h.addr, msg.GetUID()))

			if err := h.SendCot(cot.MakePong()); err != nil {
				h.logger.Error("SendMsg error", "error", err)
			}

			if !h.routePings {
				continue
			}
		}

		// pong
		if msg.GetType() == "t-x-c-t-r" {
			continue
		}

		h.messageCb(msg)
	}
}

//nolint:nilnil
func (h *ConnClientHandler) processXMLRead(er *cot.TagReader) (*cot.CotMessage, error) {
	tag, dat, err := er.ReadTag()
	if err != nil {
		return nil, err
	}

	if tag == "?xml" {
		return nil, nil
	}

	if tag == "auth" {
		// <auth><cot username=\"test\" password=\"111111\" uid=\"ANDROID-xxxx\ callsign=\"zzz\""/></auth>
		return nil, nil
	}

	if tag != "event" {
		return nil, fmt.Errorf("bad tag: %s", dat)
	}

	ev := new(cot.Event)
	if err := xml.Unmarshal(dat, ev); err != nil {
		return nil, fmt.Errorf("xml decode error: %w, client: %s", err, string(dat))
	}

	h.setActivity()

	h.logger.Debug("xml event: " + string(dat))

	if ev.Type == "t-x-takp-q" {
		ver := ev.Detail.GetFirst("TakControl").GetFirst("TakRequest").GetAttr("version")
		if ver == "1" {
			if err := h.sendEvent(cot.ProtoChangeOkMsg()); err == nil {
				h.logger.Info(fmt.Sprintf("client %s switch to v.1", h.addr))
				h.SetVersion(1)

				return nil, nil
			}

			return nil, fmt.Errorf("error on send ok: %w", err)
		}
	}

	if h.isClient && ev.Type == "t-x-takp-v" {
		if ps := ev.Detail.GetFirst("TakControl").GetFirst("TakProtocolSupport"); ps != nil {
			v := ps.GetAttr("version")
			h.logger.Info("server supports protocol v" + v)

			if v == "1" {
				h.logger.Debug("sending v1 req")
				_ = h.sendEvent(cot.VersionReqMsg(1))
			}
		} else {
			h.logger.Warn("invalid protocol support message: " + string(dat))
		}

		return nil, nil
	}

	if h.isClient && ev.Type == "t-x-takp-r" {
		if n := ev.Detail.GetFirst("TakControl").GetFirst("TakResponse"); n != nil {
			status := n.GetAttr("status")
			h.logger.Info("server switches to v1: " + status)

			if status == "true" {
				h.SetVersion(1)
			} else {
				h.logger.Error(fmt.Sprintf("got TakResponce with status %s: %s", status, ev.Detail))
			}
		}

		return nil, nil
	}

	return cot.EventToProto(ev)
}

func (h *ConnClientHandler) processProtoRead(r *cot.ProtoReader) (*cot.CotMessage, error) {
	msg, err := r.ReadProtoBuf()
	if err != nil {
		return nil, err
	}

	h.setActivity()

	var d *cot.Node
	d, err = cot.DetailsFromString(msg.GetCotEvent().GetDetail().GetXmlDetail())

	h.logger.Debug(fmt.Sprintf("proto msg: %s", msg))

	return &cot.CotMessage{TakMessage: msg, Detail: d}, err
}

func (h *ConnClientHandler) SetVersion(n int32) {
	atomic.StoreInt32(&h.ver, n)
}

func (h *ConnClientHandler) GetVersion() int32 {
	return atomic.LoadInt32(&h.ver)
}

func (h *ConnClientHandler) GetUID(callsign string) string {
	res := ""

	h.uids.Range(func(key, value any) bool {
		if callsign == value.(string) {
			res = key.(string)

			return false
		}

		return true
	})

	return res
}

func (h *ConnClientHandler) ForAllUID(fn func(string, string) bool) {
	h.uids.Range(func(key, value any) bool {
		return fn(key.(string), value.(string))
	})
}

func (h *ConnClientHandler) handleWrite() {
	for msg := range h.sendChan {
		if _, err := h.conn.Write(msg); err != nil {
			h.logger.Debug(fmt.Sprintf("client %s write error %v", h.addr, err))
			h.Stop()

			break
		}
	}
}

func (h *ConnClientHandler) Stop() {
	if atomic.CompareAndSwapInt32(&h.active, 1, 0) {
		h.logger.Info("stopping")
		h.cancel()

		close(h.sendChan)

		if h.conn != nil {
			_ = h.conn.Close()
		}

		h.removeCb(h)

		if h.closeTimer != nil {
			h.closeTimer.Stop()
		}
	}
}

func (h *ConnClientHandler) setActivity() {
	now := time.Now()
	h.lastActivity.Store(&now)

	if h.closeTimer == nil {
		h.closeTimer = time.AfterFunc(idleTimeout, h.closeIdle)
	} else {
		h.closeTimer.Reset(idleTimeout)
	}
}

func (h *ConnClientHandler) closeIdle() {
	last := h.lastActivity.Load()
	if last == nil {
		h.logger.Info("closing connection due to idle")
		_ = h.conn.Close()

		return
	}

	idle := time.Since(*last)

	if idle >= idleTimeout {
		h.logger.Info(fmt.Sprintf("closing connection due to idle timeout: %v", idle))
		_ = h.conn.Close()
	}
}

func (h *ConnClientHandler) sendEvent(evt *cot.Event) error {
	if h.GetVersion() != 0 {
		return fmt.Errorf("bad client version")
	}

	msg, err := xml.Marshal(evt)
	if err != nil {
		return err
	}

	h.logger.Debug("sending " + string(msg))

	if h.tryAddPacket(msg) {
		return nil
	}

	return fmt.Errorf("client is off")
}

func (h *ConnClientHandler) SendMsg(msg *cot.CotMessage) error {
	if msg.IsLocal() || h.CanSeeScope(msg.Scope) {
		return h.SendCot(msg.GetTakMessage())
	}

	if viper.GetBool("interscope_chat") && (msg.IsChat() || msg.IsChatReceipt()) {
		return h.SendCot(cot.CloneMessageNoCoords(msg.GetTakMessage()))
	}

	return nil
}

func (h *ConnClientHandler) SendCot(msg *cotproto.TakMessage) error {
	switch h.GetVersion() {
	case 0:
		buf, err := xml.Marshal(cot.ProtoToEvent(msg))
		if err != nil {
			return err
		}

		if h.tryAddPacket(buf) {
			return nil
		}
	case 1:
		buf, err := cot.MakeProtoPacket(msg)
		if err != nil {
			return err
		}

		if h.tryAddPacket(buf) {
			return nil
		}
	}

	return fmt.Errorf("client is off")
}

func (h *ConnClientHandler) tryAddPacket(msg []byte) bool {
	if !h.IsActive() {
		return false
	}
	select {
	case h.sendChan <- msg:
	default:
		if h.dropMetric != nil {
			h.dropMetric.WithLabelValues("reason", "handler_"+h.GetUser().GetLogin()).Inc()
		}
	}

	return true
}
