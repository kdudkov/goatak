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
	"github.com/kdudkov/goatak/model"
)

const (
	idleTimeout = 1 * time.Minute
)

type ClientHandler struct {
	conn         net.Conn
	ver          int32
	uid          string
	callsign     string
	lastActivity time.Time
	closeTimer   *time.Timer
	lastWrite    time.Time
	app          *App
	sendChan     chan []byte
	active       int32
	mx           sync.RWMutex
}

func NewClientHandler(conn net.Conn, app *App) *ClientHandler {
	c := &ClientHandler{
		conn:     conn,
		app:      app,
		ver:      0,
		sendChan: make(chan []byte, 10),
		active:   1,
		mx:       sync.RWMutex{},
	}

	return c
}

func (h *ClientHandler) Start() {
	h.app.Logger.Infof("ssl client")
	go h.handleRead()
	go h.handleWrite()

	h.AddEvent(cotxml.VersionSupportMsg(1))
}

func (h *ClientHandler) handleRead() {
	defer h.stopHandle()

	er := cot.NewTagReader(h.conn)
	pr := cot.NewProtoReader(h.conn)

	for {
		if atomic.LoadInt32(&h.active) != 1 {
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
				break
			}
			h.app.Logger.Debug(err.Error())
			break
		}

		if msg == nil {
			continue
		}

		cotmsg := &cot.Msg{
			From:       h.uid,
			TakMessage: msg,
			Detail:     d,
		}

		h.checkFirstMsg(cotmsg)

		if err != nil {
			h.app.Logger.Errorf("error decoding details: %v", err)
			continue
		}

		h.app.Logger.Debugf("details: %s", msg.GetCotEvent().GetDetail().GetXmlDetail())

		h.app.ch <- cotmsg
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
		return nil, nil, fmt.Errorf("bad tag: %s", dat)
	}

	ev := &cotxml.Event{}
	if err := xml.Unmarshal(dat, ev); err != nil {
		return nil, nil, fmt.Errorf("xml decode error: %v, data: %s", err, string(dat))
	}

	if ev.IsTakControlRequest() {
		ver := ev.Detail.TakControl.TakRequest.Version
		if ver == 1 {
			if err := h.AddEvent(cotxml.ProtoChangeOkMsg()); err == nil {
				h.app.Logger.Infof("client %s switch to v.1", h.uid)
				h.SetVersion(1)
				return nil, nil, nil
			} else {
				return nil, nil, fmt.Errorf("error on send ok: %v", err)
			}
		}
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

func (h *ClientHandler) checkFirstMsg(msg *cot.Msg) {
	if h.GetUid() == "" && msg.TakMessage.GetCotEvent() != nil {
		uid := msg.TakMessage.CotEvent.Uid
		if strings.HasSuffix(uid, "-ping") {
			uid = uid[:len(uid)-5]
		}
		h.SetUid(uid)
		h.app.AddHandler(uid, h)
	}
	if h.GetCallsign() == "" && msg.TakMessage.GetCotEvent().GetDetail().GetContact() != nil {
		h.callsign = msg.TakMessage.GetCotEvent().GetDetail().GetContact().Callsign
		h.app.AddContact(msg.TakMessage.CotEvent.Uid, model.ContactFromEvent(msg))
	}
}

func (h *ClientHandler) SetUid(uid string) {
	h.mx.Lock()
	defer h.mx.Unlock()
	h.uid = uid
}

func (h *ClientHandler) GetUid() string {
	h.mx.RLock()
	defer h.mx.RUnlock()
	return h.uid
}

func (h *ClientHandler) SetCallsign(callsign string) {
	h.mx.Lock()
	defer h.mx.Unlock()
	h.callsign = callsign
}

func (h *ClientHandler) GetCallsign() string {
	h.mx.RLock()
	defer h.mx.RUnlock()
	return h.callsign
}

func (h *ClientHandler) handleWrite() {
	for msg := range h.sendChan {
		if _, err := h.conn.Write(msg); err != nil {
			h.app.Logger.Debugf("client %s write error %v", h.uid, err)
			h.stopHandle()
			break
		}
		h.setWriteActivity()
	}
}

func (h *ClientHandler) stopHandle() {
	if atomic.CompareAndSwapInt32(&h.active, 1, 0) {
		if h.uid != "" {
			h.app.RemoveHandler(h.uid)
		}

		close(h.sendChan)

		if h.conn != nil {
			h.conn.Close()
		}

		if c := h.app.GetContact(h.uid); c != nil {
			c.SetOffline()
		}
		h.app.SendToAllOther(cot.MakeOfflineMsg(h.uid, "a-f-G"), h.uid)
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
		h.app.Logger.Debugf("closing tcp connection due to idle timeout: %v", idle)
		h.conn.Close()
	}
}

func (h *ClientHandler) setWriteActivity() {
	h.lastWrite = time.Now()
}

func (h *ClientHandler) AddEvent(evt *cotxml.Event) error {
	if atomic.LoadInt32(&h.active) != 1 {
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

func (h *ClientHandler) AddMsg(msg *cotproto.TakMessage) error {
	if atomic.LoadInt32(&h.active) != 1 {
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
	if atomic.LoadInt32(&h.active) != 1 {
		return false
	}
	select {
	case h.sendChan <- msg:
	default:
	}
	return true
}
