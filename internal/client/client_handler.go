package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/kdudkov/goatak/pkg/cot"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/kdudkov/goatak/pkg/cotproto"
)

const (
	idleTimeout = 5 * time.Minute
	pingTimeout = time.Second * 15
)

type HandlerConfig struct {
	User      string
	Scope     string
	Serial    string
	Uid       string
	IsClient  bool
	MessageCb func(msg *cot.CotMessage)
	RemoveCb  func(ch ClientHandler)
	Logger    *zap.SugaredLogger
}

type ClientHandler interface {
	GetName() string
	GetUids() map[string]string
	GetUser() string
	GetScope() string
	CanSeeScope(scope string) bool
	GetVersion() int32
	SendMsg(msg *cotproto.TakMessage) error
	GetLastSeen() *time.Time
}

type ConnClientHandler struct {
	ctx          context.Context
	cancel       context.CancelFunc
	conn         net.Conn
	addr         string
	localUid     string
	ver          int32
	isClient     bool
	uids         sync.Map
	lastActivity atomic.Pointer[time.Time]
	closeTimer   *time.Timer
	sendChan     chan []byte
	active       int32
	user         string
	scope        string
	serial       string
	messageCb    func(msg *cot.CotMessage)
	removeCb     func(ch ClientHandler)
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

	c.ctx, c.cancel = context.WithCancel(context.Background())

	if config != nil {
		c.user = config.User
		c.scope = config.Scope
		c.serial = config.Serial
		c.localUid = config.Uid
		if config.Logger != nil {
			c.logger = config.Logger.Named("client " + name)
		}
		c.isClient = config.IsClient
		c.messageCb = config.MessageCb
		c.removeCb = config.RemoveCb
	}
	c.closeTimer = time.AfterFunc(idleTimeout, c.closeIdle)
	return c
}

func (h *ConnClientHandler) GetName() string {
	return h.addr
}

func (h *ConnClientHandler) GetUser() string {
	return h.user
}

func (h *ConnClientHandler) GetScope() string {
	return h.scope
}

func (h *ConnClientHandler) CanSeeScope(scope string) bool {
	return h.scope == "" || scope == "broadcast" || h.scope == scope
}

func (h *ConnClientHandler) GetUids() map[string]string {
	res := make(map[string]string)
	h.uids.Range(func(key, value any) bool {
		res[key.(string)] = value.(string)
		return true
	})
	return res
}

func (h *ConnClientHandler) IsActive() bool {
	return atomic.LoadInt32(&h.active) == 1
}

func (h *ConnClientHandler) GetLastSeen() *time.Time {
	return h.lastActivity.Load()
}

func (h *ConnClientHandler) Start() {
	h.logger.Info("starting")
	go h.handleWrite()
	go h.handleRead()

	if h.isClient {
		go h.pinger()
	}

	if !h.isClient {
		h.logger.Infof("send version msg")
		if err := h.sendEvent(cot.VersionSupportMsg(1)); err != nil {
			h.logger.Errorf("error sending ver req: %s", err.Error())
		}
	}
}

func (h *ConnClientHandler) pinger() {
	ticker := time.NewTicker(pingTimeout)
	defer ticker.Stop()
	for h.ctx.Err() == nil {
		select {
		case <-ticker.C:
			h.logger.Debugf("ping")
			if err := h.SendMsg(cot.MakePing(h.localUid)); err != nil {
				h.logger.Debugf("sendMsg error: %v", err)
			}
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *ConnClientHandler) handleRead() {
	defer h.stopHandle()

	er := cot.NewTagReader(h.conn)
	pr := cot.NewProtoReader(h.conn)

	for h.ctx.Err() == nil {
		var msg *cotproto.TakMessage
		var d *cot.Node
		var err error

		switch h.GetVersion() {
		case 0:
			msg, d, err = h.processXMLRead(er)
		case 1:
			msg, d, err = h.processProtoRead(pr)
		}

		if err != nil {
			if err == io.EOF {
				h.logger.Info("EOF")
				break
			}
			h.logger.Warnf(err.Error())
			break
		}

		if msg == nil {
			continue
		}

		cotmsg := &cot.CotMessage{
			From:       h.addr,
			Scope:      h.scope,
			TakMessage: msg,
			Detail:     d,
		}

		// add new contact uid
		if cotmsg.IsContact() {
			uid := msg.GetCotEvent().GetUid()
			if strings.HasSuffix(uid, "-ping") {
				uid = uid[:len(uid)-5]
			}
			h.uids.Store(uid, cotmsg.GetCallsign())
		}

		// remove contact
		if cotmsg.GetType() == "t-x-d-d" && cotmsg.Detail != nil && cotmsg.Detail.Has("link") {
			uid := cotmsg.Detail.GetFirst("link").GetAttr("uid")
			h.logger.Debugf("delete uid %s by message", uid)
			h.uids.Delete(uid)
		}

		// ping
		if cotmsg.GetType() == "t-x-c-t" {
			h.logger.Debugf("ping from %s %s", h.addr, cotmsg.GetUid())
			if err := h.SendMsg(cot.MakePong()); err != nil {
				h.logger.Errorf("SendMsg error: %v", err)
			}
		}

		h.messageCb(cotmsg)
	}
}

func (h *ConnClientHandler) processXMLRead(er *cot.TagReader) (*cotproto.TakMessage, *cot.Node, error) {
	tag, dat, err := er.ReadTag()
	if err != nil {
		return nil, nil, err
	}

	if tag == "?xml" {
		return nil, nil, nil
	}

	if tag == "auth" {
		// <auth><cot username=\"test\" password=\"111111\" uid=\"ANDROID-xxxx\ callsign=\"zzz\""/></auth>
		return nil, nil, nil
	}

	if tag != "event" {
		return nil, nil, fmt.Errorf("bad tag: %s", dat)
	}

	ev := &cot.Event{}
	if err := xml.Unmarshal(dat, ev); err != nil {
		return nil, nil, fmt.Errorf("xml decode error: %v, client: %s", err, string(dat))
	}

	h.setActivity()

	h.logger.Debugf("xml event: %s", ev)

	if ev.IsTakControlRequest() {
		ver := ev.Detail.GetFirst("TakControl").GetFirst("TakRequest").GetAttr("version")
		if ver == "1" {
			if err := h.sendEvent(cot.ProtoChangeOkMsg()); err == nil {
				h.logger.Infof("client %s switch to v.1", h.addr)
				h.SetVersion(1)
				return nil, nil, nil
			} else {
				return nil, nil, fmt.Errorf("error on send ok: %v", err)
			}
		}
	}

	if h.isClient && ev.Detail.GetFirst("TakControl").Has("TakProtocolSupport") {
		v := ev.Detail.GetFirst("TakControl").GetFirst("TakProtocolSupport").GetAttr("version")
		h.logger.Infof("server supports protocol v%s", v)
		if v == "1" {
			h.logger.Debugf("sending v1 req")
			_ = h.sendEvent(cot.VersionReqMsg(1))
		}
		return nil, nil, nil
	}

	if h.isClient && ev.Detail.GetFirst("TakControl").Has("TakResponse") {
		status := ev.Detail.GetFirst("TakControl").GetFirst("TakResponse").GetAttr("status")
		h.logger.Infof("server switches to v1: %v", status)
		if status == "true" {
			h.SetVersion(1)
		} else {
			h.logger.Errorf("got TakResponce with status %s: %s", status, ev.Detail)
		}
		return nil, nil, nil
	}

	msg, d := cot.EventToProto(ev)

	return msg, d, nil
}

func (h *ConnClientHandler) processProtoRead(r *cot.ProtoReader) (*cotproto.TakMessage, *cot.Node, error) {
	msg, err := r.ReadProtoBuf()
	if err != nil {
		return nil, nil, err
	}

	h.setActivity()

	var d *cot.Node
	d, err = cot.DetailsFromString(msg.GetCotEvent().GetDetail().GetXmlDetail())

	return msg, d, err
}

func (h *ConnClientHandler) SetVersion(n int32) {
	atomic.StoreInt32(&h.ver, n)
}

func (h *ConnClientHandler) GetVersion() int32 {
	return atomic.LoadInt32(&h.ver)
}

func (h *ConnClientHandler) checkContact(msg *cot.CotMessage) {
	if msg.IsContact() {
		uid := msg.TakMessage.CotEvent.Uid
		if strings.HasSuffix(uid, "-ping") {
			uid = uid[:len(uid)-5]
		}
		h.uids.Store(uid, msg.GetCallsign())
	}

	if msg.GetType() == "t-x-d-d" && msg.Detail != nil && msg.Detail.Has("link") {
		uid := msg.Detail.GetFirst("link").GetAttr("uid")
		h.uids.Delete(uid)
	}
}

func (h *ConnClientHandler) GetUid(callsign string) string {
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

func (h *ConnClientHandler) ForAllUid(fn func(string, string) bool) {
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
	return
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
	idle := time.Now().Sub(*last)

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

func (h *ConnClientHandler) SendMsg(msg *cotproto.TakMessage) error {
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
