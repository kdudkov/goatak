package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
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
	app.AddEventProcessor("items", app.saveItemProcessor, "a-", "b-", "u-")
	app.AddEventProcessor("logger", app.loggerProcessor, ".-")

	if app.config.logging {
		app.AddEventProcessor("file_logger", app.fileLoggerProcessor, ".-")
	}
}

func (app *App) loggerProcessor(msg *cot.CotMessage) {
	if !strings.HasPrefix(msg.GetType(), "a-") {
		app.Logger.Debugf("%s %s", msg.GetType(), cot.GetMsgType(msg.GetType()))
	}
}

func (app *App) removeItemProcessor(msg *cot.CotMessage) {
	// t-x-d-d
	if link := msg.GetFirstLink("p-p"); link != nil {
		uid := link.GetAttr("uid")
		typ := link.GetAttr("type")

		if uid == "" {
			app.Logger.Warnf("invalid remove message: %s", msg.GetDetail())

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
				app.missions.DeletePoint(uid)

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
		c.From = app.items.GetCallsign(c.FromUID)
	}

	app.Logger.Infof("Chat %s", c.String())

	app.messages = append(app.messages, c)
	if err := logChatMessage(c); err != nil {
		app.Logger.Warnf("error logging chat: %s", err.Error())
	}
}

func (app *App) saveItemProcessor(msg *cot.CotMessage) {
	if cot.MatchAnyPattern(msg.GetType(), "b-t-f", "b-t-f-") {
		return
	}

	cl := model.GetClass(msg)
	if c := app.items.Get(msg.GetUID()); c != nil {
		app.Logger.Debugf("update %s %s (%s) %s", cl, msg.GetUID(), msg.GetCallsign(), msg.GetType())
		c.Update(msg)
	} else {
		app.Logger.Infof("new %s %s (%s) %s", cl, msg.GetUID(), msg.GetCallsign(), msg.GetType())
		app.items.Store(model.FromMsg(msg))
	}
}

func (app *App) fileLoggerProcessor(msg *cot.CotMessage) {
	if cot.MatchAnyPattern(msg.GetType(), "t-x-c-t", "t-x-c-t-r") {
		return
	}

	if cot.MatchAnyPattern(msg.GetType(), viper.GetStringSlice("log_exclude")...) {
		return
	}

	if err := logMessage(msg, filepath.Join(app.config.dataDir, "log")); err != nil {
		app.Logger.Warnf("error logging message: %s", err.Error())
	}
}

func logMessage(msg *cot.CotMessage, dir string) error {
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	fname := filepath.Join(dir, time.Now().Format("2006-01-02.tak"))

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

func logChatMessage(c *model.ChatMessage) error {
	fd, err := os.OpenFile("msg.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return nil
	}

	defer fd.Close()
	_, err = fmt.Fprintf(fd, "%s %s (%s) -> %s (%s) \"%s\"\n", c.Time, c.From, c.FromUID, c.Chatroom, c.ToUID, c.Text)

	return err
}
