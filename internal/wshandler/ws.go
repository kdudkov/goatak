package wshandler

import (
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/gofiber/contrib/websocket"
	"github.com/kdudkov/goatak/pkg/model"
)

type WebMessage struct {
	Typ         string             `json:"type"`
	Unit        *model.WebUnit     `json:"unit,omitempty"`
	UID         string             `json:"uid,omitempty"`
	ChatMessage *model.ChatMessage `json:"chat_msg,omitempty"`
}

type JSONWsHandler struct {
	log    *slog.Logger
	name   string
	ws     *websocket.Conn
	ch     chan *WebMessage
	active int32
}

func NewHandler(log *slog.Logger, name string, ws *websocket.Conn) *JSONWsHandler {
	return &JSONWsHandler{
		log:    log.With("client", name),
		name:   name,
		ws:     ws,
		ch:     make(chan *WebMessage, 10),
		active: 1,
	}
}

func (w *JSONWsHandler) IsActive() bool {
	return w != nil && atomic.LoadInt32(&w.active) == 1
}

func (w *JSONWsHandler) stop() {
	if atomic.CompareAndSwapInt32(&w.active, 1, 0) {
		close(w.ch)
		w.ws.Close()
	}
}

func (w *JSONWsHandler) writer() {
	for item := range w.ch {
		if !w.IsActive() {
			return
		}

		if item == nil {
			continue
		}

		_ = w.ws.WriteJSON(item)
	}
}

func (w *JSONWsHandler) reader() {
	defer w.stop()

	for {
		_, _, err := w.ws.ReadMessage()

		if err != nil {
			w.log.Error("error on read", slog.Any("error", err))

			return
		}
	}
}

func (w *JSONWsHandler) SendItem(i *model.Item) bool {
	if w == nil || !w.IsActive() {
		return false
	}

	select {
	case w.ch <- &WebMessage{Typ: "unit", Unit: i.ToWeb()}:
	default:
	}

	return true
}

func (w *JSONWsHandler) DeleteItem(uid string) bool {
	if w == nil || !w.IsActive() {
		return false
	}

	select {
	case w.ch <- &WebMessage{Typ: "delete", UID: uid}:
	default:
	}

	return true
}

func (w *JSONWsHandler) NewChatMessage(msg *model.ChatMessage) bool {
	if w == nil || !w.IsActive() {
		return false
	}

	select {
	case w.ch <- &WebMessage{Typ: "chat", ChatMessage: msg}:
	default:
	}

	return true
}

func (w *JSONWsHandler) closehandler(code int, text string) error {
	w.log.Info(fmt.Sprintf("closed with code %d, msg %s", code, text))
	w.stop()

	return nil
}

func (w *JSONWsHandler) Listen() {
	w.log.Debug("ws start")
	w.ws.SetCloseHandler(w.closehandler)

	go w.writer()
	w.reader()
	w.log.Debug("ws stop")
}
