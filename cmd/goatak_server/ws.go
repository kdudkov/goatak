package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aofei/air"
	"go.uber.org/zap"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

type WsClientHandler struct {
	name      string
	user      *model.User
	ws        *air.WebSocket
	ch        chan []byte
	uids      sync.Map
	active    int32
	messageCb func(msg *cot.CotMessage)
	logger    *zap.SugaredLogger
}

func (w *WsClientHandler) GetName() string {
	return w.name
}

func (w *WsClientHandler) GetUser() *model.User {
	return w.user
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

func NewWsClient(name string, user *model.User, ws *air.WebSocket, log *zap.SugaredLogger, mc func(msg *cot.CotMessage)) *WsClientHandler {
	return &WsClientHandler{
		name:      name,
		user:      user,
		ws:        ws,
		uids:      sync.Map{},
		ch:        make(chan []byte, 10),
		active:    1,
		logger:    log.Named(name),
		messageCb: mc,
	}
}

func (w *WsClientHandler) SendMsg(msg *cot.CotMessage) error {
	return w.SendCot(msg.TakMessage)
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
			w.logger.Errorf("send error: %s", err.Error())
			w.stop()

			break
		}
	}
}

func (w *WsClientHandler) stop() {
	if atomic.CompareAndSwapInt32(&w.active, 1, 0) {
		close(w.ch)
		w.ws.Close()
	}
}

func (w *WsClientHandler) Listen() {
	w.ws.BinaryHandler = func(b []byte) error {
		msg, err := cot.ReadProto(bufio.NewReader(bytes.NewReader(b)))
		if err != nil {
			w.logger.Errorf("read: %s", err.Error())

			return err
		}

		cotmsg, err := cot.CotFromProto(msg, w.name, w.GetUser().GetScope())
		if err != nil {
			w.logger.Errorf("details get error: %s", err.Error())

			return err
		}

		if cotmsg.IsContact() {
			uid := msg.GetCotEvent().GetUid()
			uid = strings.TrimSuffix(uid, "-ping")

			w.uids.Store(uid, cotmsg.GetCallsign())
		}

		// remove contact
		if cotmsg.GetType() == "t-x-d-d" && cotmsg.Detail != nil && cotmsg.Detail.Has("link") {
			uid := cotmsg.Detail.GetFirst("link").GetAttr("uid")
			w.uids.Delete(uid)
		}

		w.messageCb(cotmsg)

		return nil
	}
	go w.writer()
	w.ws.Listen()
	w.logger.Infof("stop listening")
}

func getWsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}

		defer ws.Close()

		app.Logger.Infof("WS connection from %s", req.ClientAddress())
		name := "ws:" + req.ClientAddress()
		w := NewWsClient(name, nil, ws, app.Logger, app.NewCotMessage)

		app.AddClientHandler(w)
		w.Listen()
		app.RemoveHandlerCb(w)
		app.Logger.Infof("ws disconnected")

		return nil
	}
}
