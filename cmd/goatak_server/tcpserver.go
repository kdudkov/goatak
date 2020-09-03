package main

import (
	"encoding/xml"
	"github.com/google/uuid"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/kdudkov/goatak/cot"
)

const (
	idleTimeout = 1 * time.Minute
)

type ClientHandler struct {
	conn         net.Conn
	Uid          string
	Callsign     string
	lastActivity time.Time
	closeTimer   *time.Timer
	lastWrite    time.Time
	app          *App
	Ch           chan []byte
	active       atomic.Bool
	mx           sync.Mutex
}

func (app *App) ListenTCP(addressPort string) (err error) {
	listen, err := net.Listen("tcp", addressPort)
	if err != nil {
		app.Logger.Errorf("Failed to listen: %v", err)
		return err
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			app.Logger.Errorf("Unable to accept connections: %#v", err)
			return err
		}

		NewClientHandler(conn, app).Start()
	}
}

func NewClientHandler(conn net.Conn, app *App) *ClientHandler {
	c := &ClientHandler{
		conn:   conn,
		app:    app,
		Ch:     make(chan []byte, 10),
		active: atomic.Bool{},
		mx:     sync.Mutex{},
	}
	c.active.Store(true)
	return c
}

func (h *ClientHandler) Start() {
	go h.handleRead()
	go h.handleWrite()
}

func (h *ClientHandler) handleRead() {
	defer h.stopHandle()

	er := cot.NewEventnReader(h.conn)

Loop:
	for {
		if !h.active.Load() {
			break
		}

		dat, err := er.ReadEvent()
		if err != nil {
			if err == io.EOF {
				break Loop
			}
			h.app.Logger.Errorf("read error: %v", err)
			continue
		}

		ev := &cot.Event{}
		if err := xml.Unmarshal(dat, ev); err != nil {
			h.app.Logger.Errorf("decode error: %v, data: %s", err, string(dat))
			continue
		}

		h.checkFirstMsg(ev)
	}

	if h.closeTimer != nil {
		h.closeTimer.Stop()
	}
}

func (h *ClientHandler) checkFirstMsg(evt *cot.Event) {
	if strings.HasPrefix(evt.Type, "a-f-") {
		// position (assume it's client one)
		if h.Uid == "" {
			h.Uid = evt.Uid
			h.Callsign = evt.GetCallsign()
			h.app.AddClient(evt.Uid, h)
		}
	}
}

func (h *ClientHandler) processEvent(dat []byte, evt *cot.Event) {
	h.app.ch <- &Msg{dat: dat, event: evt}
}

func (h *ClientHandler) handleWrite() {
	for {
		msg := <-h.Ch

		if _, err := h.conn.Write(msg); err != nil {
			h.stopHandle()
			break
		}
		h.setWriteActivity()
	}
}

func (h *ClientHandler) stopHandle() {
	h.mx.Lock()
	defer h.mx.Unlock()

	if h.active.CAS(true, false) {
		if h.Uid != "" {
			h.app.RemoveClient(h.Uid)
		}

		close(h.Ch)

		if h.conn != nil {
			h.conn.Close()
		}

		h.app.SendToAll(MakeOfflineMsg(h.Uid, "a-f-G"), h.Uid)
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

func (h *ClientHandler) AddMsg(msg []byte) bool {
	h.mx.Lock()
	defer h.mx.Unlock()

	if h.active.Load() {
		select {
		case h.Ch <- msg:
		default:
		}
		return true
	}
	return false
}

func MakeOfflineMsg(uid string, typ string) *cot.Event {
	evt := cot.BasicEvent("t-x-d-d", uuid.New().String(), time.Minute*3)
	evt.How = "h-g-i-g-o"
	evt.Detail = cot.Detail{Link: []*cot.Link{{Uid: uid, Type: typ, Relation: "p-p"}}}
	return evt
}
