package cot

import (
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
	"github.com/kdudkov/goatak/cotproto"
	"github.com/kdudkov/goatak/cotxml"
	"go.uber.org/zap"
)

const (
	idleTimeout = 1 * time.Minute
	pingTimeout = time.Second * 15
)

type HandlerConfig struct {
	User      string
	Uid       string
	IsClient  bool
	MessageCb func(msg *Msg)
	RemoveCb  func(ch *ClientHandler)
	Logger    *zap.SugaredLogger
}

type ClientHandler struct {
	conn         net.Conn
	addr         string
	uid          string
	ver          int32
	isClient     bool
	uids         sync.Map
	lastActivity time.Time
	closeTimer   *time.Timer
	sendChan     chan []byte
	active       int32
	user         string
	messageCb    func(msg *Msg)
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

	if config != nil {
		c.user = config.User
		c.uid = config.Uid
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
		_ = h.SendEvent(cotxml.VersionSupportMsg(1))
	}
}

func (h *ClientHandler) pinger() {
	for {
		if !h.IsActive() {
			return
		}

		time.Sleep(pingTimeout)
		h.logger.Debugf("ping")
		if err := h.SendMsg(MakePing(h.uid)); err != nil {
			h.logger.Debugf("sendMsg error: %v", err)
		}
	}
}

func (h *ClientHandler) handleRead() {
	defer h.stopHandle()

	er := NewTagReader(h.conn)
	pr := NewProtoReader(h.conn)

	for {
		if !h.IsActive() {
			break
		}

		var msg *cotproto.TakMessage
		var d *XMLDetails
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

		cotmsg := &Msg{
			From:       h.addr,
			TakMessage: msg,
			Detail:     d,
		}

		h.checkContact(cotmsg)

		if cotmsg.GetType() == "t-x-c-t-r" {
			h.logger.Debug("pong")
			continue
		}

		h.messageCb(cotmsg)

		if h.closeTimer != nil {
			h.closeTimer.Stop()
		}
	}
}

func (h *ClientHandler) processXMLRead(er *TagReader) (*cotproto.TakMessage, *XMLDetails, error) {
	tag, dat, err := er.ReadTag()
	if err != nil {
		return nil, nil, err
	}

	if tag == "?xml" {
		return nil, nil, nil
	}

	if tag != "event" {
		//return nil, nil, fmt.Errorf("bad tag: %s", dat)
		return nil, nil, nil
	}

	ev := &cotxml.Event{}
	if err := xml.Unmarshal(dat, ev); err != nil {
		return nil, nil, fmt.Errorf("xml decode error: %v, data: %s", err, string(dat))
	}

	if ev.IsTakControlRequest() {
		ver := ev.Detail.TakControl.TakRequest.Version
		if ver == 1 {
			if err := h.SendEvent(cotxml.ProtoChangeOkMsg()); err == nil {
				h.logger.Infof("client %s switch to v.1", h.addr)
				h.SetVersion(1)
				return nil, nil, nil
			} else {
				return nil, nil, fmt.Errorf("error on send ok: %v", err)
			}
		}
	}

	if h.isClient && ev.Detail.TakControl != nil && ev.Detail.TakControl.TakProtocolSupport != nil {
		v := ev.Detail.TakControl.TakProtocolSupport.Version
		h.logger.Infof("server supports protocol v%d", v)
		if v >= 1 {
			_ = h.SendEvent(cotxml.VersionReqMsg(1))
		}
		return nil, nil, nil
	}

	if h.isClient && ev.Detail.TakControl != nil && ev.Detail.TakControl.TakResponce != nil {
		ok := ev.Detail.TakControl.TakResponce.Status
		h.logger.Infof("server switches to v1: %v", ok)
		if ok {
			h.SetVersion(1)
		}
		return nil, nil, nil
	}

	msg, d := EventToProto(ev)

	return msg, d, nil
}

func (h *ClientHandler) processProtoRead(r *ProtoReader) (*cotproto.TakMessage, *XMLDetails, error) {
	buf, err := r.ReadProtoBuf()
	if err != nil {
		return nil, nil, err
	}

	msg := new(cotproto.TakMessage)
	if err := proto.Unmarshal(buf, msg); err != nil {
		return nil, nil, fmt.Errorf("failed to decode protobuf: %v", err)
	}

	var d *XMLDetails
	d, err = DetailsFromString(msg.GetCotEvent().GetDetail().GetXmlDetail())

	return msg, d, err
}

func (h *ClientHandler) SetVersion(n int32) {
	atomic.StoreInt32(&h.ver, n)
}

func (h *ClientHandler) GetVersion() int32 {
	return atomic.LoadInt32(&h.ver)
}

func (h *ClientHandler) checkContact(msg *Msg) {
	if msg.IsContact() {
		uid := msg.TakMessage.CotEvent.Uid
		if strings.HasSuffix(uid, "-ping") {
			uid = uid[:len(uid)-5]
		}
		h.uids.Store(uid, msg.GetCallsign())
	}

	if msg.GetType() == "t-x-d-d" && msg.Detail != nil && msg.Detail.HasChild("link") {
		uid := msg.Detail.GetFirstChild("link").GetAttr("uid")
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
		close(h.sendChan)

		if h.conn != nil {
			_ = h.conn.Close()
		}

		h.removeCb(h)
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

func (h *ClientHandler) SendEvent(evt *cotxml.Event) error {
	if !h.IsActive() {
		return nil
	}

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
	if !h.IsActive() {
		return nil
	}

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
