package client

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

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
	Logger       *zap.SugaredLogger
}

type ClientHandler interface {
	GetName() string
	HasUID(uid string) bool
	GetUids() map[string]string
	GetUser() *model.User
	GetVersion() int32
	SendMsg(msg *cot.CotMessage) error
	SendCot(msg *cotproto.TakMessage) error
	GetLastSeen() *time.Time
}

type ConnClientHandler struct {
	cancel       context.CancelFunc
	conn         net.Conn
	addr         string
	localUID     string
	ver          int32
	isClient     bool
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
	logger       *zap.SugaredLogger
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

		if config.Logger != nil {
			c.logger = config.Logger.Named("client " + name)
		}

		c.isClient = config.IsClient
		c.messageCb = config.MessageCb
		c.removeCb = config.RemoveCb
		c.newContactCb = config.NewContactCb
	}

	c.closeTimer = time.AfterFunc(idleTimeout, c.closeIdle)

	return c
}

func (h *ConnClientHandler) GetName() string {
	return h.addr
}

func (h *ConnClientHandler) GetUser() *model.User {
	return h.user
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
		h.logger.Debugf("send version msg")

		if err := h.sendEvent(cot.VersionSupportMsg(1)); err != nil {
			h.logger.Errorf("error sending ver req: %s", err.Error())
		}
	}
}

func (h *ConnClientHandler) pinger(ctx context.Context) {
	ticker := time.NewTicker(pingTimeout)
	defer ticker.Stop()

	for ctx.Err() == nil {
		select {
		case <-ticker.C:
			h.logger.Debugf("ping")

			if err := h.SendCot(cot.MakePing(h.localUID)); err != nil {
				h.logger.Debugf("sendMsg error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *ConnClientHandler) handleRead(ctx context.Context) {
	defer h.stopHandle()

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

			h.logger.Warnf(err.Error())

			break
		}

		if msg == nil {
			continue
		}

		msg.From = h.addr
		msg.Scope = h.getScope()

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
			h.logger.Debugf("delete uid %s by message", uid)
			h.uids.Delete(uid)
		}

		// ping
		if msg.GetType() == "t-x-c-t" {
			h.logger.Debugf("ping from %s %s", h.addr, msg.GetUID())

			if err := h.SendCot(cot.MakePong()); err != nil {
				h.logger.Errorf("SendMsg error: %v", err)
			}
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

	h.logger.Debugf("xml event: %s", dat)

	if ev.Type == "t-x-takp-q" {
		ver := ev.Detail.GetFirst("TakControl").GetFirst("TakRequest").GetAttr("version")
		if ver == "1" {
			if err := h.sendEvent(cot.ProtoChangeOkMsg()); err == nil {
				h.logger.Infof("client %s switch to v.1", h.addr)
				h.SetVersion(1)

				return nil, nil
			}

			return nil, fmt.Errorf("error on send ok: %w", err)
		}
	}

	if h.isClient && ev.Type == "t-x-takp-v" {
		if ps := ev.Detail.GetFirst("TakControl").GetFirst("TakProtocolSupport"); ps != nil {
			v := ps.GetAttr("version")
			h.logger.Infof("server supports protocol v%s", v)

			if v == "1" {
				h.logger.Debugf("sending v1 req")
				_ = h.sendEvent(cot.VersionReqMsg(1))
			}
		} else {
			h.logger.Warnf("invalid protocol support message: %s", dat)
		}

		return nil, nil
	}

	if h.isClient && ev.Type == "t-x-takp-r" {
		if n := ev.Detail.GetFirst("TakControl").GetFirst("TakResponse"); n != nil {
			status := n.GetAttr("status")
			h.logger.Infof("server switches to v1: %v", status)

			if status == "true" {
				h.SetVersion(1)
			} else {
				h.logger.Errorf("got TakResponce with status %s: %s", status, ev.Detail)
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

	h.logger.Debugf("proto msg: %s", msg)

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
			h.logger.Debugf("client %s write error %v", h.addr, err)
			h.stopHandle()

			break
		}
	}
}

func (h *ConnClientHandler) stopHandle() {
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
		h.logger.Infof("closing connection due to idle")
		_ = h.conn.Close()

		return
	}

	idle := time.Since(*last)

	if idle >= idleTimeout {
		h.logger.Infof("closing connection due to idle timeout: %v", idle)
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

	h.logger.Debugf("sending %s", msg)

	if h.tryAddPacket(msg) {
		return nil
	}

	return fmt.Errorf("client is off")
}

func (h *ConnClientHandler) SendMsg(msg *cot.CotMessage) error {
	if h.GetUser().CanSeeScope(msg.Scope) {
		return h.SendCot(msg.TakMessage)
	}

	if msg.IsChat() || msg.IsChatReceipt() {
		return h.SendCot(cot.CloneMessageNoCoords(msg.TakMessage))
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
	}

	return true
}

func (h *ConnClientHandler) getScope() string {
	if h.user == nil {
		return ""
	}

	return h.user.Scope
}
