package main

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jroimartin/gocui"

	"github.com/kdudkov/goatak/internal/model"
)

const (
	missionsView = "missions"
	missionView  = "mission"
)

type binding struct {
	view string
	key  gocui.Key
	mod  gocui.Modifier
	f    func(_ *gocui.Gui, _ *gocui.View) error
}

func (app *App) setBindings() error {
	bindings := []binding{
		{"", gocui.KeyCtrlC, gocui.ModNone, app.stop},
		{missionsView, gocui.KeyArrowUp, gocui.ModNone, app.cursorUp},
		{missionsView, gocui.KeyArrowDown, gocui.ModNone, app.cursorDown},
	}

	for _, b := range bindings {
		if err := app.g.SetKeybinding(b.view, b.key, b.mod, b.f); err != nil {
			return err
		}
	}

	return nil
}

func (app *App) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView(missionsView, 0, 0, maxX/2-1, maxY-1); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		v.Frame = true
		v.Highlight = true
		v.Title = "Missions"
		v.BgColor = gocui.ColorBlack | gocui.AttrBold
		v.SelBgColor = gocui.ColorWhite
	}

	if v, err := g.SetView(missionView, maxX/2, 0, maxX-1, maxY-1); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		v.Frame = true
		v.Title = "Mission details"
	}

	_, err := g.SetCurrentView(missionsView)
	app.drawMission()

	return err
}

func (app *App) redraw() {
	if app.g == nil {
		return
	}

	app.g.Update(func(gui *gocui.Gui) error {
		if v, err := gui.View(missionsView); err == nil {
			v.Clear()
			res := make([]*model.MissionDTO, 0)

			app.missions.Range(func(key, value any) bool {
				res = append(res, value.(*model.MissionDTO))

				return true
			})

			sort.Slice(res, func(i, j int) bool {
				return res[i].Name < res[j].Name
			})

			for _, u := range res {
				fmt.Fprintf(v, "%s\n", u.Name)
			}
		}

		if v, err := gui.View(missionView); err == nil {
			v.Clear()
		}

		return nil
	})
}

func (app *App) cursorUp(g *gocui.Gui, v *gocui.View) error {
	v.MoveCursor(0, -1, false)
	app.drawMission()

	return nil
}

func (app *App) cursorDown(g *gocui.Gui, v *gocui.View) error {
	v.MoveCursor(0, 1, false)
	app.drawMission()

	return nil
}

func (app *App) drawMission() {
	var name string

	if v, err := app.g.View(missionsView); err == nil {
		_, y := v.Cursor()
		l, _ := v.Line(y)
		name = l
	}

	if v, err := app.g.View(missionView); err == nil {
		v.Clear()

		if name == "" {
			fmt.Fprintf(v, "no mission")
			return
		}

		if val, ok := app.missions.Load(name); ok {
			if m, ok1 := val.(*model.MissionDTO); ok1 {
				fmt.Fprintf(v, "Name: %s\n", m.Name)
				fmt.Fprintf(v, "Description: %s\n", m.Description)
				fmt.Fprintf(v, "Created: %s by %s\n", ft(m.CreateTime), m.CreatorUID)

				if len(m.Uids) > 0 {
					fmt.Fprintf(v, "\nPoints (%d):\n", len(m.Uids))

					for _, c := range m.Uids {
						fmt.Fprintf(v, "%s %s %s\n", c.Data, c.Details.Type, c.Details.Callsign)
					}
				}

				if len(m.Contents) > 0 {
					fmt.Fprintf(v, "\nContent (%d):\n", len(m.Contents))

					for _, c := range m.Contents {
						fmt.Fprintf(v, "%s %s %d\n", c.Data.Name, c.Data.MimeType, c.Data.Size)
					}
				}
			}
		}

		//s, err := app.remoteAPI.GetSubscriptions(context.Background(), name)
		//
		//if err != nil {
		//	s = []string{err.Error()}
		//}
		//fmt.Fprintf(v, strings.Join(s, ","))
		//
		//s1, err := app.remoteAPI.GetSubscriptionRoles(context.Background(), name)
		//fmt.Fprintf(v, s1)
	}
}

func ft(t model.CotTime) string {
	t1 := time.Time(t)

	if t1.IsZero() {
		return ""
	}

	return t1.Format("02-01-2006 15:04")
}
