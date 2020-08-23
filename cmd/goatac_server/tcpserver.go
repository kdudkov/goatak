package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"

	"gotac/cot"
	"gotac/xml"
)

const (
	idleTimeout = 5 * time.Minute
	pingTimeout = 5 * time.Second

	debug = true
)

type ClientHandler struct {
	conn         net.Conn
	Uid          string
	Callsign     string
	lastActivity time.Time
	closeTimer   *time.Timer
	lastWrite    time.Time
	pingTimer    *time.Timer
	app          *App
	Ch           chan []byte
	log          *os.File
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

	var dec *xml.Decoder
	if debug {
		h.log, _ = os.Create(time.Now().Format("20060102-15040507.log"))
		dec = xml.NewDecoder(io.TeeReader(h.conn, h.log))
	} else {
		dec = xml.NewDecoder(h.conn)
	}

Loop:
	for {
		if !h.active.Load() {
			break
		}
		// Read tokens from the XML document in a stream.
		t, _ := dec.Token()
		if t == nil {
			h.app.Logger.Infof("stop reading for %s", h.Uid)
			break
		}
		h.setActivity()

		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "event" {
				var ev cot.Event
				if err := dec.DecodeElement(&ev, &se); err != nil {
					h.app.Logger.Errorf("error decoding element: %v", err)
					continue
				}
				if err := h.processEvent(&ev); err != nil {
					h.app.Logger.Errorf("%v", err)
					break Loop
				}
			}
		case xml.CharData:
		case xml.ProcInst:
			continue
		default:
			h.app.Logger.Errorf("wtf? %s\n", reflect.TypeOf(t).Name())
		}
	}

	if h.closeTimer != nil {
		h.closeTimer.Stop()
	}
}

func (h *ClientHandler) processEvent(evt *cot.Event) error {
	if strings.HasPrefix(evt.Type, "a-f-") {
		// position (assume it's client one)
		if h.Uid == "" {
			h.Uid = evt.Uid
			h.app.AddClient(evt.Uid, h)
		} else {
			if h.Uid != evt.Uid {
				return fmt.Errorf("bad uid: was %s, now %s", h.Uid, evt.Uid)
			}
			h.Callsign = evt.GetCallsign()
		}
	}
	h.app.ch <- evt
	return nil
}

func (h *ClientHandler) handleWrite() {
	for {
		msg := <-h.Ch

		if _, err := h.conn.Write(msg); err != nil {
			h.app.Logger.Infof("stop writing for %s", h.Uid)
			if h.pingTimer != nil {
				h.pingTimer.Stop()
			}
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
		close(h.Ch)
		if h.log != nil {
			h.log.Close()
		}
		if h.Uid != "" {
			h.app.RemoveClient(h.Uid)
		}

		if h.conn != nil {
			h.conn.Close()
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

// closeIdle closes the connection if last activity is passed behind idleTimeout.
func (h *ClientHandler) closeIdle() {
	idle := time.Now().Sub(h.lastActivity)

	if idle >= idleTimeout {
		h.app.Logger.Debugf("closing tcp connection due to idle timeout: %v", idle)
		h.conn.Close()
	}
}

func (h *ClientHandler) setWriteActivity() {
	h.lastWrite = time.Now()

	if h.pingTimer == nil {
		h.pingTimer = time.AfterFunc(pingTimeout, h.sendPing)
	} else {
		h.pingTimer.Reset(pingTimeout)
	}
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

func (h *ClientHandler) sendPing() {
	if time.Now().Sub(h.lastWrite) > pingTimeout {
		if msg, err := xml.Marshal(cot.MakePing(h.app.uid)); err == nil {
			h.AddMsg(msg)
		}
	}
}
