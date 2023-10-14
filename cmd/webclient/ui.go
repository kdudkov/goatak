package main

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
	"sort"
	"strings"
)

func (app *App) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("info", 0, 0, maxX/2-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = true
	}
	if v, err := g.SetView("log", maxX/2, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = true
		v.Title = "Log"
	}
	return nil
}

func (app *App) redraw() {
	if app.g == nil {
		return
	}

	app.g.Update(func(gui *gocui.Gui) error {
		if v, err := gui.View("info"); err == nil {
			v.Clear()
			if app.IsConnected() {
				fmt.Fprintf(v, WithColors("Connected to %s:%s as %s\n\n", FgGreen, Bold), app.host, app.tcpPort, app.callsign)
			} else {
				fmt.Fprintf(v, WithColors("Disconnected\n\n", FgWhite))
			}

			res := make([]*model.WebUnit, 0)
			app.items.ForEach(func(i *model.Item) bool {
				if i.GetClass() == model.CONTACT {
					res = append(res, i.ToWeb())
				}
				return true
			})

			sort.Slice(res, func(i, j int) bool {
				return res[i].Callsign < res[j].Callsign
			})

			for _, u := range res {
				if u.Status == "Online" {
					fmt.Fprintf(v, WithColors("%s %s %s [%s] %.5f,%.5f\n", FgGreen, Bold), u.Callsign, u.Team, u.Role, u.Status, u.Lat, u.Lon)
				} else {
					fmt.Fprintf(v, WithColors("%s %s %s [%s]\n", FgWhite), u.Callsign, u.Team, u.Role, u.Status)
				}
			}
		}
		if v, err := gui.View("log"); err == nil {
			v.Clear()
			_, size := v.Size()
			for _, l := range app.textLogger.GetLines(size) {
				fmt.Fprintln(v, l)
			}
		}

		return nil
	})
}

func (app *App) LogMessage(msg *cot.CotMessage) {
	var col []byte
	var extra string

	switch {
	case msg.GetType() == "t-x-c-t":
		extra = "ping"
		col = []byte{FgBlack, Bold}
	case msg.GetType() == "t-x-c-t-r":
		extra = "pong"
		col = []byte{FgBlack, Bold}
	case msg.GetType() == "t-x-d-d":
		extra = "remove msg"
		if msg.Detail != nil && msg.Detail.Has("link") {
			uid := msg.Detail.GetFirst("link").GetAttr("uid")
			typ := msg.Detail.GetFirst("link").GetAttr("type")
			extra += fmt.Sprintf(" uid: %s, type %s", uid, typ)
			if uid != "" {
				if i := app.items.Get(uid); i != nil {
					extra += " callsign " + i.GetCallsign()
				}
			}
		} else {
			extra += " " + msg.TakMessage.GetCotEvent().GetDetail().GetXmlDetail()
		}
		col = []byte{FgRed}
	case msg.IsChat():
		if c := model.MsgToChat(msg); c != nil {
			extra = c.String()
		} else {
			extra = "invalid chat message"
		}
		col = []byte{FgYellow, Bold}
		break
	case msg.IsChatReceipt():
		extra = "chat receipt"
		col = []byte{FgYellow, Bold}
		break
	case strings.HasPrefix(msg.GetType(), "a-"):
		extra = msg.GetCallsign()
		switch msg.GetType()[2] {
		case 'f':
			col = []byte{FgBlue, Bold}
		case 'h':
			col = []byte{FgRed, Bold}
		case 'n':
			col = []byte{FgGreen, Bold}
		default:
			col = []byte{FgWhite, Bold}
		}
	case strings.HasPrefix(msg.GetType(), "b-"):
		extra = msg.GetCallsign()
		col = []byte{FgCyan, Bold}
	case strings.HasPrefix(msg.GetType(), "u-"):
		col = []byte{FgMagenta, Bold}
	case msg.GetType() == "tak registration":
		extra = ""
	default:
		extra = "unknown"
	}

	app.textLogger.AddLineColor(fmt.Sprintf("%s %s", msg.GetType(), extra), col...)
	app.redraw()
}
