package main

import (
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
	"google.golang.org/protobuf/proto"
	"os"
	"strings"
)

type EventProcessor struct {
	include []string
	cb      func(msg *cot.CotMessage)
}

func (app *App) AddEventProcessor(name string, cb func(msg *cot.CotMessage), masks ...string) {
	app.eventProcessors[name] = &EventProcessor{cb: cb, include: masks}
}

func (app *App) InitMessageProcessors() {
	app.AddEventProcessor("remove", app.removeItemProcessor, "t-x-d-d")
	app.AddEventProcessor("chat", app.chatProcessor, "b-t-f-")
	app.AddEventProcessor("items", app.saveItemProcessor, "a-", "b-")
	app.AddEventProcessor("logger", app.loggerProcessor, "-")

	if app.saveFile != "" {
		app.AddEventProcessor("file_logger", app.fileLoggerProcessor, "-")
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
		name, exact := cot.GetMsgType(msg.GetType())
		if exact {
			app.Logger.Debugf("%s %s", msg.GetType(), name)
		} else {
			app.Logger.Infof("%s %s (extended)", msg.GetType(), name)
		}
	}
}

func (app *App) removeItemProcessor(msg *cot.CotMessage) {
	// t-x-d-d
	if msg.Detail != nil && msg.Detail.Has("link") {
		uid := msg.Detail.GetFirst("link").GetAttr("uid")
		typ := msg.Detail.GetFirst("link").GetAttr("type")
		if uid == "" {
			app.Logger.Warnf("invalid remove message: %s", msg.Detail)
			return
		}
		if v := app.items.Get(uid); v != nil {
			switch v.GetClass() {
			case model.CONTACT:
				app.Logger.Debugf("remove %s by message", uid)
				v.SetOffline()
				app.processChange(v)
				return
			case model.UNIT, model.POINT:
				app.Logger.Debugf("remove unit/point %s type %s by message", uid, typ)
				app.items.Remove(uid)
				return
			}
		}
	}
}

func (app *App) chatProcessor(msg *cot.CotMessage) {
	c := model.MsgToChat(msg)
	if c == nil {
		app.Logger.Errorf("invalid chat message %s", msg.TakMessage)
		return
	}
	if c.From == "" {
		c.From = app.items.GetCallsign(c.FromUid)
	}
	app.Logger.Infof("%s", c)
	app.messages.Add(c)
}

func (app *App) saveItemProcessor(msg *cot.CotMessage) {
	if cot.MatchPattern(msg.GetType(), "b-t-f-") {
		return
	}

	if msg.GetUid() == app.uid {
		app.Logger.Debugf("my own position")
	}

	cl := model.GetClass(msg)
	if c := app.items.Get(msg.GetUid()); c != nil {
		app.Logger.Debugf("update %s %s (%s) %s", cl, msg.GetUid(), msg.GetCallsign(), msg.GetType())
		c.Update(msg)
	} else {
		app.Logger.Infof("new %s %s (%s) %s", cl, msg.GetUid(), msg.GetCallsign(), msg.GetType())
		app.items.Store(model.FromMsg(msg))
	}
}

func (app *App) fileLoggerProcessor(msg *cot.CotMessage) {
	if app.saveFile == "" {
		return
	}
	if cot.MatchAnyPattern(msg.GetType(), "t-x-c-t", "t-x-c-t-r") {
		return
	}
	if err := logMessage(msg, app.saveFile); err != nil {
		app.Logger.Warnf("error logging message: %s", err.Error())
	}
}

func logMessage(msg *cot.CotMessage, fname string) error {
	// don't save pings
	if msg.GetType() == "t-x-c-t" || msg.GetType() == "t-x-c-t-r" {
		return nil
	}

	f, err := os.OpenFile(fname, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	d, err := proto.Marshal(msg.TakMessage)
	if err != nil {
		return err
	}
	l := uint32(len(d))
	_, _ = f.Write([]byte{byte(l % 256), byte(l / 256)})
	_, _ = f.Write(d)
	return nil
}
