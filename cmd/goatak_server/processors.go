package main

import (
	"encoding/json"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
	"os"
	"strings"
)

func (app *App) InitMessageProcessors() {
	app.eventProcessors["t-x-d-d"] = app.removeItemProcessor
	// ping
	app.eventProcessors["t-x-c-t"] = app.justLogProcessor
	// pong
	app.eventProcessors["t-x-c-t-r"] = app.justLogProcessor
	// chat
	app.eventProcessors["b-t-f"] = app.chatProcessor
	app.eventProcessors["b-t-f-"] = app.logInterestingProcessor
	app.eventProcessors["a-"] = app.ProcessItem
	app.eventProcessors["b-"] = app.ProcessItem
	app.eventProcessors["u-"] = app.logInterestingProcessor
}

func (app *App) GetProcessor(t string) (string, EventProcessor) {
	var found string
	for k, v := range app.eventProcessors {
		if k == t {
			return k, v
		}
		if strings.HasSuffix(k, "-") && len(k) > len(found) && strings.HasPrefix(t, k) {
			found = k
		}
	}

	if found != "" {
		return found, app.eventProcessors[found]
	}

	return "", nil
}

func (app *App) justLogProcessor(msg *cot.CotMessage) {
	app.Logger.Debugf("%s %s", msg.GetType(), msg.GetUid())
}

func (app *App) logInterestingProcessor(msg *cot.CotMessage) {
	if app.config.logging {
		if err := logToFile(msg); err != nil {
			app.Logger.Errorf("%v", err)
		}
	}
	app.Logger.Infof("%s %s", msg.GetType(), msg.GetUid())
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
		c.From = app.GetCallsign(c.FromUid)
	}
	app.Logger.Infof("Chat %s (%s) -> %s (%s) \"%s\"", c.From, c.FromUid, c.Chatroom, c.ToUid, c.Text)
	app.messages = append(app.messages, c)
	app.logMessage(c)
}

func (app *App) ProcessItem(msg *cot.CotMessage) {
	cl := model.GetClass(msg)
	if c := app.items.Get(msg.GetUid()); c != nil {
		app.Logger.Debugf("update %s %s (%s) %s", cl, msg.GetUid(), msg.GetCallsign(), msg.GetType())
		c.Update(msg)
	} else {
		app.Logger.Infof("new %s %s (%s) %s", cl, msg.GetUid(), msg.GetCallsign(), msg.GetType())
		app.items.Store(model.FromMsg(msg))
	}
}

func logToFile(msg *cot.CotMessage) error {
	f, err := os.OpenFile(msg.GetType()+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(msg.TakMessage)
	if err != nil {
		return err
	}
	_, _ = f.WriteString(string(b))
	_, _ = f.WriteString("\n")
	return nil
}
