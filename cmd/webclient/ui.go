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
				return res[i].Callsign > res[j].Callsign
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
	switch {
	case msg.GetType() == "t-x-c-t":
		app.textLogger.AddLine(fmt.Sprintf("%s ping from %s", msg.GetType(), msg.GetUid()))
	case msg.GetType() == "t-x-c-t-r":
		app.textLogger.AddLine(fmt.Sprintf("%s pong from %s", msg.GetType(), msg.GetUid()))
	case msg.GetType() == "t-x-d-d":
		app.textLogger.AddLineColor(fmt.Sprintf("%s remove msg", msg.GetType()), FgRed)
	case msg.IsChat():
		if c := model.MsgToChat(msg); c != nil {
			app.textLogger.AddLineColor(fmt.Sprintf("%s chat msg %s (%s)-> %s room %s", msg.GetType(),
				c.From, c.FromUid, c.ToUid, c.Chatroom), FgYellow, Bold)
		} else {
			app.textLogger.AddLineColor(fmt.Sprintf("%s invalid chat msg", msg.GetType()), FgYellow, Bold)
		}
		break
	case msg.IsChatReceipt():
		app.Logger.Infof("got receipt %s", msg.GetType())
		break
	case strings.HasPrefix(msg.GetType(), "a-"):
		var col []byte
		switch msg.GetType()[2] {
		case 'f':
			col = []byte{FgBlue, Bold}
		case 'h':
			col = []byte{FgRed, Bold}
		case 'n':
			col = []byte{FgGreen, Bold}
		default:
			col = []byte{FgWhite}
		}
		app.textLogger.AddLineColor(fmt.Sprintf("%s %s", msg.GetType(), msg.GetCallsign()), col...)
	case strings.HasPrefix(msg.GetType(), "b-"):
		app.textLogger.AddLineColor(fmt.Sprintf("%s %s", msg.GetType(), msg.GetCallsign()), BgCyan)
	case strings.HasPrefix(msg.GetType(), "u-"):
		app.textLogger.AddLine(fmt.Sprintf("%s %s", msg.GetType(), msg.GetCallsign()))
	case msg.GetType() == "tak registration":
		app.textLogger.AddLine(fmt.Sprintf("%s %s", msg.GetType(), msg.GetCallsign()))
	default:
		app.textLogger.AddLine(fmt.Sprintf("%s ???", msg.GetType()))
	}
	app.redraw()
}
