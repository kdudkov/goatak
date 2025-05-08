package tak_ws

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/contrib/websocket"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/kdudkov/goatak/pkg/model"
)

type MessageCb func(msg *cot.CotMessage)

type WsClientHandler struct {
	log       *slog.Logger
	name      string
	user      *model.Device
	ws        *websocket.Conn
	ch        chan []byte
	uids      sync.Map
	active    int32
	messageCb MessageCb
}

func New(name string, user *model.Device, ws *websocket.Conn, mc MessageCb) *WsClientHandler {
	return &WsClientHandler{
		log:       slog.Default().With("logger", "tak_ws", "name", name, "user", user.GetLogin()),
		name:      name,
		user:      user,
		ws:        ws,
		uids:      sync.Map{},
		ch:        make(chan []byte, 10),
		active:    1,
		messageCb: mc,
	}
}

func (w *WsClientHandler) GetName() string {
	return w.name
}

func (w *WsClientHandler) GetDevice() *model.Device {
	return w.user
}

func (w *WsClientHandler) GetSerial() string {
	return ""
}

func (w *WsClientHandler) GetVersion() int32 {
	return 1
}

func (w *WsClientHandler) GetUids() map[string]string {
	res := make(map[string]string)

	w.uids.Range(func(key, value any) bool {
		res[key.(string)] = value.(string)

		return true
	})

	return res
}

func (w *WsClientHandler) HasUID(uid string) bool {
	_, ok := w.uids.Load(uid)

	return ok
}

func (w *WsClientHandler) HasCallsign(callsign string) bool {
	var found bool

	w.uids.Range(func(_, value any) bool {
		if value.(string) == callsign {
			found = true
			return false
		}

		return true
	})

	return found
}

func (w *WsClientHandler) GetLastSeen() *time.Time {
	return nil
}

func (w *WsClientHandler) SendMsg(msg *cot.CotMessage) error {
	if msg.IsLocal() || w.user.CanSeeScope(msg.Scope) {
		return w.SendCot(msg.GetTakMessage())
	}

	return nil
}

func (w *WsClientHandler) SendCot(msg *cotproto.TakMessage) error {
	dat, err := cot.MakeProtoPacket(msg)
	if err != nil {
		return err
	}

	if w.tryAddPacket(dat) {
		return nil
	}

	return fmt.Errorf("client is off")
}

func (w *WsClientHandler) tryAddPacket(msg []byte) bool {
	if !w.IsActive() {
		return false
	}
	select {
	case w.ch <- msg:
	default:
	}

	return true
}

func (w *WsClientHandler) IsActive() bool {
	return atomic.LoadInt32(&w.active) == 1
}

func (w *WsClientHandler) writer() {
	for b := range w.ch {
		if err := w.ws.WriteMessage(websocket.BinaryMessage, b); err != nil {
			w.log.Error("send error", slog.Any("error", err))
			w.Stop()

			break
		}
	}
}

func (w *WsClientHandler) reader() {
	defer w.Stop()

	for {
		_ = w.ws.SetReadDeadline(time.Now().Add(time.Minute))
		mt, b, err := w.ws.ReadMessage()

		if err != nil {
			w.log.Error("read error", slog.Any("error", err))

			return
		}

		if mt != websocket.BinaryMessage {
			continue
		}

		if err = w.parse(b); err != nil {
			w.log.Error("parse", slog.Any("error", err))
		}
	}
}

func (w *WsClientHandler) Stop() {
	if atomic.CompareAndSwapInt32(&w.active, 1, 0) {
		close(w.ch)
		_ = w.ws.Close()
	}
}

func (w *WsClientHandler) Listen() {
	go w.writer()
	w.reader()
	w.log.Info("stop listening")
}

func (w *WsClientHandler) parse(b []byte) error {
	msg, err := cot.ReadProto(bufio.NewReader(bytes.NewReader(b)))
	if err != nil {
		return fmt.Errorf("read error %w", err)
	}

	cotmsg, err := cot.CotFromProto(msg, w.name, w.GetDevice().GetScope())
	if err != nil {
		return fmt.Errorf("convert error %w", err)
	}

	if cotmsg.IsContact() {
		uid := msg.GetCotEvent().GetUid()
		uid = strings.TrimSuffix(uid, "-ping")

		w.uids.Store(uid, cotmsg.GetCallsign())
	}

	// remove contact
	if cotmsg.GetType() == "t-x-d-d" && cotmsg.GetDetail() != nil && cotmsg.GetDetail().Has("link") {
		uid := cotmsg.GetDetail().GetFirst("link").GetAttr("uid")
		w.uids.Delete(uid)
	}

	w.messageCb(cotmsg)

	return nil
}
