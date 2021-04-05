package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	v0 "github.com/kdudkov/goatak/cot/v0"
	v1 "github.com/kdudkov/goatak/cot/v1"
	"github.com/kdudkov/goatak/model"

	"github.com/kdudkov/goatak/cot"
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

func (app *App) ListenTCP(addr string) (err error) {
	app.Logger.Infof("listening TCP at %s", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		app.Logger.Errorf("Failed to listen: %v", err)
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			app.Logger.Errorf("Unable to accept connections: %#v", err)
			return err
		}

		NewClientHandler(conn, app).Start()
	}
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
	go h.handleRead()
	go h.handleWrite()

	h.AddEvent(v0.VersionMsg(1))
}

func (h *ClientHandler) handleRead() {
	defer h.stopHandle()

	er := cot.NewTagReader(h.conn)
	pr := cot.NewProtoReader(h.conn)

	for {
		if atomic.LoadInt32(&h.active) != 1 {
			break
		}

		var msg *v1.TakMessage
		var err error

		switch h.GetVersion() {
		case 0:
			msg, err = h.processXMLRead(er)
		case 1:
			msg, err = h.processProtoRead(pr)
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			h.app.Logger.Debug(err.Error())
			continue
		}

		if msg == nil {
			continue
		}

		h.checkFirstMsg(msg)
		h.processEvent(msg)
	}

	if h.closeTimer != nil {
		h.closeTimer.Stop()
	}
}

func (h *ClientHandler) processXMLRead(er *cot.TagReader) (*v1.TakMessage, error) {
	tag, dat, err := er.ReadTag()
	if err != nil {
		return nil, err
	}

	if tag == "?xml" {
		return nil, nil
	}

	if tag != "event" {
		return nil, fmt.Errorf("bad tag: %s", dat)
	}

	ev := &v0.Event{}
	if err := xml.Unmarshal(dat, ev); err != nil {
		return nil, fmt.Errorf("xml decode error: %v, data: %s", err, string(dat))
	}

	if ev.IsTakControlRequest() {
		ver := ev.Detail.TakControl.TakRequest.Version
		if ver == 1 {
			if err := h.AddEvent(v0.ProtoChangeOkMsg()); err == nil {
				h.app.Logger.Infof("client %s switch to v.1", h.uid)
				h.SetVersion(1)
				return nil, nil
			} else {
				return nil, fmt.Errorf("error on send ok: %v", err)
			}
		}
	}

	msg, _ := cot.EventToProto(ev)
	return msg, nil
}

func (h *ClientHandler) processProtoRead(r *cot.ProtoReader) (*v1.TakMessage, error) {
	buf, err := r.ReadProtoBuf()
	if err != nil {
		return nil, err
	}

	msg := new(v1.TakMessage)
	if err := proto.Unmarshal(buf, msg); err != nil {

		return nil, fmt.Errorf("failed to decode protobuf: %v", err)
	}

	if msg.GetCotEvent().GetDetail().GetXmlDetail() != "" {
		h.app.Logger.Debugf("%s", msg.CotEvent.Detail.XmlDetail)
	}

	return msg, nil
}

func (h *ClientHandler) SetVersion(n int32) {
	atomic.StoreInt32(&h.ver, n)
}

func (h *ClientHandler) GetVersion() int32 {
	return atomic.LoadInt32(&h.ver)
}

func (h *ClientHandler) checkFirstMsg(msg *v1.TakMessage) {
	if h.GetUid() == "" && msg.GetCotEvent() != nil {
		h.SetUid(msg.CotEvent.Uid)
		h.app.AddHandler(msg.CotEvent.Uid, h)
	}
	if h.GetCallsign() == "" && msg.GetCotEvent().GetDetail().GetContact() != nil {
		h.callsign = msg.GetCotEvent().GetDetail().GetContact().Callsign
		h.app.AddContact(msg.CotEvent.Uid, model.ContactFromEvent(msg))
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

func (h *ClientHandler) processEvent(msg *v1.TakMessage) {
	d, err := v1.FromString(msg.GetCotEvent().GetDetail().GetXmlDetail())

	if err != nil {
		h.app.Logger.Errorf("error decoding details: %v", err)
		return
	}

	h.app.ch <- &v1.Msg{
		TakMessage: msg,
		Detail:     d,
	}
}

func (h *ClientHandler) handleWrite() {
	for {
		msg := <-h.sendChan

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

func (h *ClientHandler) AddEvent(evt *v0.Event) error {
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

func (h *ClientHandler) AddMsg(msg *v1.TakMessage) error {
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
		n := PutUvarint(buf, uint64(len(buf1)), 1)
		copy(buf[n:], buf1)
		if h.tryAddPacket(buf[:n+len(buf1)+1]) {
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

func PutUvarint(buf []byte, x uint64, idx int) int {
	i := idx
	for x >= 0x80 {
		buf[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	buf[i] = byte(x)
	return i + 1
}
