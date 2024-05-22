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

	"github.com/aofei/air"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

type MessageCb func(msg *cot.CotMessage)

type WsClientHandler struct {
	name      string
	user      *model.User
	ws        *air.WebSocket
	ch        chan []byte
	uids      sync.Map
	active    int32
	messageCb MessageCb
	logger    *slog.Logger
}

func (w *WsClientHandler) GetName() string {
	return w.name
}

func (w *WsClientHandler) GetUser() *model.User {
	return w.user
}

func (w *WsClientHandler) GetSerial() string {
	return ""
}

func (w *WsClientHandler) CanSeeScope(scope string) bool {
	return true
}

func (w *WsClientHandler) GetVersion() int32 {
	return 0
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

func (w *WsClientHandler) GetLastSeen() *time.Time {
	return nil
}

func New(name string, user *model.User, ws *air.WebSocket, mc MessageCb) *WsClientHandler {
	return &WsClientHandler{
		name:      name,
		user:      user,
		ws:        ws,
		uids:      sync.Map{},
		ch:        make(chan []byte, 10),
		active:    1,
		logger:    slog.Default().With("logger", "tak_ws", "name", name, "user", user),
		messageCb: mc,
	}
}

func (w *WsClientHandler) SendMsg(msg *cot.CotMessage) error {
	return w.SendCot(msg.GetTakMessage())
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
		if err := w.ws.WriteBinary(b); err != nil {
			w.logger.Error("send error", "error", err.Error())
			w.Stop()

			break
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
	w.ws.BinaryHandler = w.binaryReader
	go w.writer()
	w.ws.Listen()
	w.logger.Info("stop listening")
	w.Stop()
}

func (w *WsClientHandler) binaryReader(b []byte) error {
	msg, err := cot.ReadProto(bufio.NewReader(bytes.NewReader(b)))
	if err != nil {
		w.logger.Error("read error", "error", err.Error())

		return err
	}

	cotmsg, err := cot.CotFromProto(msg, w.name, w.GetUser().GetScope())
	if err != nil {
		w.logger.Error("defaults get error", "error", err.Error())

		return err
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
