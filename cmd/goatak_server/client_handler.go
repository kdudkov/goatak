package main

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
	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/cotproto"
	"github.com/kdudkov/goatak/cotxml"
	"go.uber.org/zap"
)

const (
	idleTimeout = 1 * time.Minute
)

type ClientHandler struct {
	conn         net.Conn
	addr         string
	ver          int32
	isClient     bool
	uids         map[string]string
	lastActivity time.Time
	closeTimer   *time.Timer
	lastWrite    time.Time
	sendChan     chan []byte
	active       int32
	ssl          bool
	user         string
	mx           sync.RWMutex
	cotProcessor func(msg *cot.Msg)
	removeCb     func(ch *ClientHandler)
	logger       *zap.SugaredLogger
}

func NewClientHandler(conn net.Conn, user string, logger *zap.SugaredLogger, fn func(msg *cot.Msg), removeCb func(ch *ClientHandler)) *ClientHandler {
	c := &ClientHandler{
		addr:         "tcp:" + conn.RemoteAddr().String(),
		conn:         conn,
		ver:          0,
		sendChan:     make(chan []byte, 10),
		active:       1,
		ssl:          false,
		user:         user,
		uids:         make(map[string]string),
		mx:           sync.RWMutex{},
		logger:       logger.Named("client tcp:" + conn.RemoteAddr().String()),
		cotProcessor: fn,
		removeCb:     removeCb,
	}

	return c
}

func NewSSLClientHandler(conn net.Conn, user string, logger *zap.SugaredLogger, fn func(msg *cot.Msg), removeCb func(ch *ClientHandler)) *ClientHandler {
	c := &ClientHandler{
		addr:         "ssl:" + conn.RemoteAddr().String(),
		conn:         conn,
		ver:          0,
		sendChan:     make(chan []byte, 10),
		active:       1,
		ssl:          true,
		user:         user,
		uids:         make(map[string]string),
		mx:           sync.RWMutex{},
		logger:       logger.Named("client ssl:" + conn.RemoteAddr().String()),
		cotProcessor: fn,
		removeCb:     removeCb,
	}

	return c
}

func (h *ClientHandler) IsActive() bool {
	return atomic.LoadInt32(&h.active) == 1
}

func (h *ClientHandler) Start() {
	go h.handleWrite()
	go h.handleRead()

	if !h.isClient {
		h.logger.Infof("send version msg")
		_ = h.SendEvent(cotxml.VersionSupportMsg(1))
	}
}

func (h *ClientHandler) handleRead() {
	defer h.stopHandle()

	er := cot.NewTagReader(h.conn)
	pr := cot.NewProtoReader(h.conn)

	for {
		if !h.IsActive() {
			break
		}

		var msg *cotproto.TakMessage
		var d *cot.XMLDetails
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

		cotmsg := &cot.Msg{
			From:       h.addr,
			TakMessage: msg,
			Detail:     d,
		}

		h.checkContact(cotmsg)

		if err != nil {
			h.logger.Errorf("error decoding details: %v", err)
			continue
		}

		h.cotProcessor(cotmsg)
	}

	if h.closeTimer != nil {
		h.closeTimer.Stop()
	}
}

func (h *ClientHandler) processXMLRead(er *cot.TagReader) (*cotproto.TakMessage, *cot.XMLDetails, error) {
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

	msg, d := cot.EventToProto(ev)

	return msg, d, nil
}

func (h *ClientHandler) processProtoRead(r *cot.ProtoReader) (*cotproto.TakMessage, *cot.XMLDetails, error) {
	buf, err := r.ReadProtoBuf()
	if err != nil {
		return nil, nil, err
	}

	msg := new(cotproto.TakMessage)
	if err := proto.Unmarshal(buf, msg); err != nil {
		return nil, nil, fmt.Errorf("failed to decode protobuf: %v", err)
	}

	var d *cot.XMLDetails
	d, err = cot.DetailsFromString(msg.GetCotEvent().GetDetail().GetXmlDetail())

	return msg, d, err
}

func (h *ClientHandler) SetVersion(n int32) {
	atomic.StoreInt32(&h.ver, n)
}

func (h *ClientHandler) GetVersion() int32 {
	return atomic.LoadInt32(&h.ver)
}

func (h *ClientHandler) checkContact(msg *cot.Msg) {
	if msg.IsContact() {

		uid := msg.TakMessage.CotEvent.Uid
		if strings.HasSuffix(uid, "-ping") {
			uid = uid[:len(uid)-5]
		}

		h.AddUid(uid, msg.GetCallsign())
	}
}

func (h *ClientHandler) AddUid(uid, callsign string) {
	h.mx.Lock()
	defer h.mx.Unlock()
	h.uids[uid] = callsign
}

func (h *ClientHandler) GetUid(callsign string) string {
	h.mx.RLock()
	defer h.mx.RUnlock()

	for k, v := range h.uids {
		if v == callsign {
			return k
		}
	}

	return ""
}

func (h *ClientHandler) ForAllUid(fn func(string, string)) {
	h.mx.RLock()
	defer h.mx.RUnlock()

	for k, v := range h.uids {
		fn(k, v)
	}
}

func (h *ClientHandler) handleWrite() {
	for msg := range h.sendChan {
		if _, err := h.conn.Write(msg); err != nil {
			h.logger.Debugf("client %s write error %v", h.addr, err)
			h.stopHandle()
			break
		}
		h.setWriteActivity()
	}
}

func (h *ClientHandler) stopHandle() {
	if atomic.CompareAndSwapInt32(&h.active, 1, 0) {
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
		h.logger.Debugf("closing tcp connection due to idle timeout: %v", idle)
		h.conn.Close()
	}
}

func (h *ClientHandler) setWriteActivity() {
	h.lastWrite = time.Now()
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
		buf, err := xml.Marshal(cot.ProtoToEvent(msg))
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
