package main

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/kdudkov/goatak/model"
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
				fmt.Fprintf(v, "Connected to %s:%s as %s\n", app.host, app.tcpPort, app.callsign)
			} else {
				fmt.Fprintf(v, "Disconnected\n")
			}

			app.items.ForEach(func(i *model.Item) bool {
				if i.GetClass() == model.CONTACT {
					u := i.ToWeb()
					fmt.Fprintf(v, "%s %s %s [%s]\n", u.Callsign, u.Team, u.Role, u.Status)
				}
				return true
			})

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
