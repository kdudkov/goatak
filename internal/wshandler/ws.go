package wshandler

import (
	"encoding/json"
	"github.com/aofei/air"
	"github.com/kdudkov/goatak/pkg/model"
	"sync/atomic"
)

type WebMessage struct {
	Typ  string         `json:"type"`
	Unit *model.WebUnit `json:"unit,omitempty"`
	UID  string         `json:"uid,omitempty"`
}

type JSONWsHandler struct {
	name   string
	ws     *air.WebSocket
	ch     chan *WebMessage
	active int32
}

func NewHandler(name string, ws *air.WebSocket) *JSONWsHandler {
	return &JSONWsHandler{
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
	defer w.stop()

	for item := range w.ch {
		if w.ws.Closed {
			return
		}

		if item == nil {
			continue
		}

		if b, err := json.Marshal(item); err == nil {
			if w.ws.WriteText(string(b)) != nil {
				return
			}
		} else {
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

func (w *JSONWsHandler) Listen() {
	if w.ws.Closed {
		return
	}

	defer w.stop()

	go w.writer()
	w.ws.Listen()
}
