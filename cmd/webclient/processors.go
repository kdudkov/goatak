package main

import (
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
)

type EventProcessor struct {
	name    string
	include []string
	cb      func(msg *cot.CotMessage)
}

func (app *App) AddEventProcessor(name string, cb func(msg *cot.CotMessage), masks ...string) {
	app.eventProcessors = append(app.eventProcessors, &EventProcessor{name: name, cb: cb, include: masks})
}

func (app *App) InitMessageProcessors() {
	app.AddEventProcessor("remove", app.removeItemProcessor, "t-x-d-d")
	app.AddEventProcessor("chat", app.chatProcessor, "b-t-f")
	app.AddEventProcessor("chat_r", app.chatReceiptProcessor, "b-t-f-")
	app.AddEventProcessor("items", app.saveItemProcessor, "a-")
	app.AddEventProcessor("logger", app.loggerProcessor, ".-")

	if app.saveFile != "" {
		app.AddEventProcessor("file_logger", app.fileLoggerProcessor, ".-")
	}
	// u-rb-a Range & Bearing – Line
	// u-r-b-c-c R&b - Circle
	// u-d-c-c Drawing Shapes – Circle
	// u-d-r Drawing Shapes – Rectangle
	// u-d-f Drawing Shapes - Free Form
	// u-d-c-e Drawing Shapes – Ellipse
}

func (app *App) loggerProcessor(msg *cot.CotMessage) {
	if !strings.HasPrefix(msg.GetType(), "a-") {
		app.logger.Debug(fmt.Sprintf("%s %s", msg.GetType(), cot.GetMsgType(msg.GetType())))
	}
}

func (app *App) removeItemProcessor(msg *cot.CotMessage) {
	// t-x-d-d
	if link := msg.GetFirstLink("p-p"); link != nil {
		uid := link.GetAttr("uid")
		typ := link.GetAttr("type")

		if uid == "" {
			app.logger.Warn("invalid remove message: " + msg.GetDetail().String())

			return
		}

		if v := app.items.Get(uid); v != nil {
			switch v.GetClass() {
			case model.CONTACT:
				app.logger.Debug(fmt.Sprintf("remove %s by message", uid))
				v.SetOffline()
				app.changeCb.AddMessage(v)

				return
			case model.UNIT, model.POINT:
				app.logger.Debug(fmt.Sprintf("remove unit/point %s type %s by message", uid, typ))
				app.items.Remove(uid)
				app.deleteCb.AddMessage(uid)

				return
			}
		}
	}
}

func (app *App) chatProcessor(msg *cot.CotMessage) {
	c := model.MsgToChat(msg)
	if c == nil {
		app.logger.Error("invalid chat message " + msg.GetTakMessage().String())

		return
	}

	if c.From == "" {
		c.From = app.items.GetCallsign(c.FromUID)
	}

	app.logger.Info(c.String())
	app.messages.Add(c)
}

func (app *App) chatReceiptProcessor(msg *cot.CotMessage) {
}

func (app *App) saveItemProcessor(msg *cot.CotMessage) {
	if msg.GetLat() == 0 && msg.GetLon() == 0 {
		return
	}

	if msg.GetUID() == app.uid {
		app.logger.Debug("my own position")

		return
	}

	cl := model.GetClass(msg)
	if c := app.items.Get(msg.GetUID()); c != nil {
		app.logger.Debug(fmt.Sprintf("update %s %s (%s) %s", cl, msg.GetUID(), msg.GetCallsign(), msg.GetType()))
		c.Update(msg)
		app.items.Store(c)
		app.changeCb.AddMessage(c)
	} else {
		app.logger.Info(fmt.Sprintf("new %s %s (%s) %s", cl, msg.GetUID(), msg.GetCallsign(), msg.GetType()))
		item := model.FromMsg(msg)
		app.items.Store(item)
		app.changeCb.AddMessage(item)
	}
}

func (app *App) fileLoggerProcessor(msg *cot.CotMessage) {
	if app.saveFile == "" {
		return
	}

	if msg.IsPing() {
		return
	}

	if err := logMessage(msg, app.saveFile); err != nil {
		app.logger.Warn("error logging message", "error", err.Error())
	}
}

func logMessage(msg *cot.CotMessage, fname string) error {
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	d, err := proto.Marshal(msg.GetTakMessage())
	if err != nil {
		return err
	}

	l := uint32(len(d))
	_, _ = f.Write([]byte{byte(l % 256), byte(l / 256)})
	_, _ = f.Write(d)

	return nil
}
