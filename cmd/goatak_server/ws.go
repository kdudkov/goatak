package main

import (
	"encoding/json"
	"sync/atomic"

	"github.com/aofei/air"
	"github.com/google/uuid"

	"github.com/kdudkov/goatak/pkg/model"
)

type JSONWsHandler struct {
	name   string
	ws     *air.WebSocket
	ch     chan *model.Item
	active int32
}

func NewHandler(name string, ws *air.WebSocket) *JSONWsHandler {
	return &JSONWsHandler{
		name:   name,
		ws:     ws,
		ch:     make(chan *model.Item, 10),
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

		if b, err := json.Marshal(item.ToWeb()); err == nil {
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
	case w.ch <- i:
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

func getWsHandler(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}

		name := uuid.New().String()

		h := NewHandler(name, ws)
		app.Logger.Info("ws listener connected")
		app.changeCb.AddCallback(name, h.SendItem)
		h.Listen()
		app.Logger.Info("ws listener disconnected")
		app.changeCb.RemoveCallback(name)

		return nil
	}
}
