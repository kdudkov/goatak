package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/proto"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
)

type EventProcessor struct {
	name    string
	include []string
	cb      func(msg *cot.CotMessage) bool
}

func (app *App) AddEventProcessor(name string, cb func(msg *cot.CotMessage) bool, masks ...string) {
	app.eventProcessors = append(app.eventProcessors, &EventProcessor{name: name, cb: cb, include: masks})
}

func (app *App) InitMessageProcessors() {
	app.AddEventProcessor("logger", app.loggerProcessor, ".-")

	if app.config.logging {
		app.AddEventProcessor("file_logger", app.fileLoggerProcessor, ".-")
	}

	app.AddEventProcessor("metrics", app.metricsProcessor, "t-x-c-m")
	app.AddEventProcessor("remove", app.removeItemProcessor, "t-x-d-d")
	app.AddEventProcessor("chat", app.chatProcessor, "b-t-f")
	app.AddEventProcessor("items", app.saveItemProcessor, "a-", "b-", "u-")
	app.AddEventProcessor("filter_control", filterProcessor, "t-")

	app.AddEventProcessor("router", app.route, ".-")
}

func (app *App) processMessage(msg *cot.CotMessage) {
	for _, prc := range app.eventProcessors {
		if cot.MatchAnyPattern(msg.GetType(), prc.include...) {
			app.logger.Debug("msg is processed by " + prc.name)

			if !prc.cb(msg) {
				app.logger.Debug("process is stopped by " + prc.name)

				return
			}
		}
	}
}

func (app *App) loggerProcessor(msg *cot.CotMessage) bool {
	if !strings.HasPrefix(msg.GetType(), "a-") {
		app.logger.Debug(fmt.Sprintf("%s %s", msg.GetType(), cot.GetMsgType(msg.GetType())))
	}

	return true
}

func (app *App) removeItemProcessor(msg *cot.CotMessage) bool {
	// t-x-d-d
	if link := msg.GetFirstLink("p-p"); link != nil {
		uid := link.GetAttr("uid")
		typ := link.GetAttr("type")

		if uid == "" {
			app.logger.Warn("invalid remove message: " + msg.GetDetail().String())

			return true
		}

		if v := app.items.Get(uid); v != nil {
			switch v.GetClass() {
			case model.CONTACT:
				app.logger.Debug(fmt.Sprintf("remove %s by message", uid))
				v.SetOffline()
				app.changeCb.AddMessage(v)
			case model.UNIT, model.POINT:
				app.logger.Debug(fmt.Sprintf("remove unit/point %s type %s by message", uid, typ))
				app.items.Remove(uid)
				app.deleteCb.AddMessage(uid)
			}
		}
	}

	return true
}

func (app *App) chatProcessor(msg *cot.CotMessage) bool {
	c := model.MsgToChat(msg)
	if c == nil {
		app.logger.Error("invalid chat message " + msg.GetTakMessage().String())

		return true
	}

	if c.From == "" {
		c.From = app.items.GetCallsign(c.FromUID)
	}

	app.logger.Info("Chat " + c.String())

	app.messages = append(app.messages, c)
	if err := logChatMessage(c); err != nil {
		app.logger.Warn("error logging chat", "error", err.Error())
	}

	return true
}

func (app *App) saveItemProcessor(msg *cot.CotMessage) bool {
	if !msg.IsMapItem() {
		return true
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

		if cl == model.CONTACT && viper.GetString("welcome_msg") != "" {
			chat := &model.ChatMessage{
				ID:       uuid.NewString(),
				Time:     time.Now(),
				Parent:   "RootContactGroup",
				Chatroom: item.GetCallsign(),
				From:     "",
				FromUID:  "ADMIN_UID",
				ToUID:    item.GetUID(),
				Direct:   true,
				Text:     viper.GetString("welcome_msg"),
			}

			app.NewCotMessage(cot.LocalCotMessage(model.MakeChatMessage(chat)))
		}
	}

	return true
}

func (app *App) fileLoggerProcessor(msg *cot.CotMessage) bool {
	if msg.IsPing() || cot.MatchAnyPattern(msg.GetType(), viper.GetStringSlice("log_exclude")...) {
		return true
	}

	if err := logMessage(msg, filepath.Join(app.config.dataDir, "log")); err != nil {
		app.logger.Warn("error logging message", "error", err.Error())
	}

	return true
}

func (app *App) metricsProcessor(msg *cot.CotMessage) bool {
	uid := msg.GetFirstLink("p-s").GetAttr("uid")

	if uid == "" {
		app.logger.Warn("no uid " + msg.GetTakMessage().GetCotEvent().GetDetail().GetXmlDetail())

		return false
	}

	if c := app.items.Get(uid); c != nil {
		if c.GetClass() != model.CONTACT {
			app.logger.Warn("got metrics for " + c.GetClass())

			return false
		}

		c.SetOnline()

		if st := msg.GetDetail().GetFirst("stats"); st != nil {
			stats := st.GetAttrs()
			app.logger.Info(fmt.Sprintf("stats for %s: %s", uid, stats))
		}
	}

	return false
}

func filterProcessor(msg *cot.CotMessage) bool {
	return !msg.IsControl()
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

	d, err := proto.Marshal(msg.GetTakMessage())
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
