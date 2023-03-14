package cot

import (
	"context"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/kdudkov/goatak/cotproto"
)

const (
	idleTimeout = 1 * time.Minute
	pingTimeout = time.Second * 15
)

type HandlerConfig struct {
	User      string
	Serial    string
	Uid       string
	IsClient  bool
	MessageCb func(msg *CotMessage)
	RemoveCb  func(ch *ClientHandler)
	Logger    *zap.SugaredLogger
}

type ClientHandler struct {
	ctx          context.Context
	cancel       context.CancelFunc
	conn         net.Conn
	addr         string
	localUid     string
	ver          int32
	isClient     bool
	uids         sync.Map
	lastActivity time.Time
	closeTimer   *time.Timer
	sendChan     chan []byte
	active       int32
	user         string
	serial       string
	messageCb    func(msg *CotMessage)
	removeCb     func(ch *ClientHandler)
	logger       *zap.SugaredLogger
}

func NewClientHandler(name string, conn net.Conn, config *HandlerConfig) *ClientHandler {
	c := &ClientHandler{
		addr:     name,
		conn:     conn,
		ver:      0,
		sendChan: make(chan []byte, 10),
		active:   1,
		uids:     sync.Map{},
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	if config != nil {
		c.user = config.User
		c.serial = config.Serial
		c.localUid = config.Uid
		if config.Logger != nil {
			c.logger = config.Logger.Named("client " + name)
		}
		c.isClient = config.IsClient
		c.messageCb = config.MessageCb
		c.removeCb = config.RemoveCb
	}
	return c
}

func (h *ClientHandler) GetName() string {
	return h.addr
}

func (h *ClientHandler) GetUser() string {
	return h.user
}

func (h *ClientHandler) GetUids() map[string]string {
	res := make(map[string]string)
	h.uids.Range(func(key, value interface{}) bool {
		res[key.(string)] = value.(string)
		return true
	})
	return res
}

func (h *ClientHandler) IsActive() bool {
	return atomic.LoadInt32(&h.active) == 1
}

func (h *ClientHandler) Start() {
	h.logger.Info("starting")
	go h.handleWrite()
	go h.handleRead()

	if h.isClient {
		go h.pinger()
	}

	if !h.isClient {
		h.logger.Infof("send version msg")
		_ = h.SendEvent(VersionSupportMsg(1))
	}
}

func (h *ClientHandler) pinger() {
	ticker := time.NewTicker(pingTimeout)
	defer ticker.Stop()
	for h.ctx.Err() == nil {
		select {
		case <-ticker.C:
			h.logger.Debugf("ping")
			if err := h.SendMsg(MakePing(h.localUid)); err != nil {
				h.logger.Debugf("sendMsg error: %v", err)
			}
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *ClientHandler) handleRead() {
	defer h.stopHandle()

	er := NewTagReader(h.conn)
	pr := NewProtoReader(h.conn)

	for h.ctx.Err() == nil {
		var msg *cotproto.TakMessage
		var d *Node
		var err error

		switch h.GetVersion() {
		case 0:
			msg, d, err = h.processXMLRead(er)
		case 1:
			msg, d, err = h.processProtoRead(pr)
		}

		h.setActivity()

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

		cotmsg := &CotMessage{
			From:       h.addr,
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
			h.uids.Delete(uid)
		}

		// ping
		if cotmsg.GetType() == "t-x-c-t" {
			if err := h.SendMsg(MakePong()); err != nil {
				h.logger.Errorf("SendMsg error: %v", err)
			}
		}

		h.messageCb(cotmsg)
	}
}

func (h *ClientHandler) processXMLRead(er *TagReader) (*cotproto.TakMessage, *Node, error) {
	tag, dat, err := er.ReadTag()
	if err != nil {
		return nil, nil, err
	}

	if tag == "?xml" {
		return nil, nil, nil
	}

	if tag != "event" {
		return nil, nil, fmt.Errorf("bad tag: %s", dat)
	}

	ev := &Event{}
	if err := xml.Unmarshal(dat, ev); err != nil {
		return nil, nil, fmt.Errorf("xml decode error: %v, data: %s", err, string(dat))
	}

	if ev.IsTakControlRequest() {
		ver := ev.Detail.GetFirst("TakControl").GetFirst("TakRequest").GetAttr("version")
		if ver == "1" {
			if err := h.SendEvent(ProtoChangeOkMsg()); err == nil {
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
			_ = h.SendEvent(VersionReqMsg(1))
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

	msg, d := EventToProto(ev)

	return msg, d, nil
}

func (h *ClientHandler) processProtoRead(r *ProtoReader) (*cotproto.TakMessage, *Node, error) {
	buf, err := r.ReadProtoBuf()
	if err != nil {
		return nil, nil, err
	}

	msg := new(cotproto.TakMessage)
	if err := proto.Unmarshal(buf, msg); err != nil {
		return nil, nil, fmt.Errorf("failed to decode protobuf: %v", err)
	}

	var d *Node
	d, err = DetailsFromString(msg.GetCotEvent().GetDetail().GetXmlDetail())

	return msg, d, err
}

func (h *ClientHandler) SetVersion(n int32) {
	atomic.StoreInt32(&h.ver, n)
}

func (h *ClientHandler) GetVersion() int32 {
	return atomic.LoadInt32(&h.ver)
}

func (h *ClientHandler) checkContact(msg *CotMessage) {
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

func (h *ClientHandler) GetUid(callsign string) string {
	res := ""
	h.uids.Range(func(key, value interface{}) bool {
		if callsign == value.(string) {
			res = key.(string)
			return false
		}
		return true
	})

	return res
}

func (h *ClientHandler) ForAllUid(fn func(string, string) bool) {
	h.uids.Range(func(key, value interface{}) bool {
		return fn(key.(string), value.(string))
	})
}

func (h *ClientHandler) handleWrite() {
	for msg := range h.sendChan {
		if _, err := h.conn.Write(msg); err != nil {
			h.logger.Debugf("client %s write error %v", h.addr, err)
			h.stopHandle()
			break
		}
	}
}

func (h *ClientHandler) stopHandle() {
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

func (h *ClientHandler) setActivity() {
	h.lastActivity = time.Now()

	if h.closeTimer == nil {
		h.closeTimer = time.AfterFunc(idleTimeout, h.closeIdle)
	} else {
		h.closeTimer.Reset(idleTimeout)
	}
}

func (h *ClientHandler) closeIdle() {
	idle := time.Now().Sub(h.lastActivity)

	if idle >= idleTimeout {
		h.logger.Infof("closing connection due to idle timeout: %v", idle)
		_ = h.conn.Close()
	}
}

func (h *ClientHandler) SendEvent(evt *Event) error {
	if h.GetVersion() != 0 {
		return fmt.Errorf("bad client version")
	}

	msg, err := xml.Marshal(evt)
	if err != nil {
		return err
	}

	if h.tryAddPacket(msg) {
		return nil
	}

	return fmt.Errorf("client is off")
}

func (h *ClientHandler) SendMsg(msg *cotproto.TakMessage) error {
	switch h.GetVersion() {
	case 0:
		buf, err := xml.Marshal(ProtoToEvent(msg))
		if err != nil {
			return err
		}
		if h.tryAddPacket(buf) {
			return nil
		}
	case 1:
		buf1, err := proto.Marshal(msg)
		if err != nil {
			return err
		}

		buf := make([]byte, len(buf1)+5)
		buf[0] = 0xbf
		n := binary.PutUvarint(buf[1:], uint64(len(buf1)))
		copy(buf[n+1:], buf1)
		if h.tryAddPacket(buf[:n+len(buf1)+2]) {
			return nil
		}
	}

	return fmt.Errorf("client is off")
}

func (h *ClientHandler) tryAddPacket(msg []byte) bool {
	if !h.IsActive() {
		return false
	}
	select {
	case h.sendChan <- msg:
	default:
	}
	return true
}
