package main

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aofei/air"
	"go.uber.org/zap"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
)

func addMissionApi(app *App, api *air.Air) {
	g := api.Group("/Marti/api/missions")

	g.GET("", getMissionsHandler(app))

	g.GET("/all/invitations", getMissionsInvitationsHandler(app))

	g.GET("/:missionname", getMissionHandler(app))
	g.PUT("/:missionname", getMissionPutHandler(app))
	g.DELETE("/:missionname", getMissionDeleteHandler(app))
	g.GET("/:missionname/changes", getMissionChangesHandler(app))
	g.GET("/:missionname/cot", getMissionCotHandler(app))
	g.GET("/:missionname/contacts", getMissionContactsHandler(app))
	g.DELETE("/:missionname/contents", getMissionContentDeleteHandler(app))
	g.GET("/:missionname/log", getMissionLogHandler(app))
	g.PUT("/:missionname/keywords", getMissionKeywordsPutHandler(app))
	g.GET("/:missionname/role", getMissionRoleHandler(app))
	g.PUT("/:missionname/role", getMissionRolePutHandler(app))
	g.GET("/:missionname/subscription", getMissionSubscriptionHandler(app))
	g.PUT("/:missionname/subscription", getMissionSubscriptionPutHandler(app))
	g.DELETE("/:missionname/subscription", getMissionSubscriptionDeleteHandler(app))
	g.GET("/:missionname/subscriptions", getMissionSubscriptionsHandler(app))
	g.GET("/:missionname/subscriptions/roles", getMissionSubscriptionRolesHandler(app))
	g.PUT("/:missionname/invite/:type/:uid", getInvitePutHandler(app))
	g.DELETE("/:missionname/invite/:type/:uid", getInviteDeleteHandler(app))
}

func getMissionsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		data := app.missions.GetAllMissions()
		result := make([]*model.MissionDTO, len(data))

		for i, m := range data {
			result[i] = model.ToMissionDTO(m)
		}

		return res.WriteJSON(makeAnswer("Mission", result))
	}
}

func getMissionHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		name := getStringParam(req, "missionname")
		m := app.missions.GetMission(name)

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		return res.WriteJSON(makeAnswer("Mission", []*model.MissionDTO{model.ToMissionDTO(m)}))
	}
}

func getMissionPutHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		username := getUsernameFromReq(req)

		printParams(req, app.Logger)

		if req.Body != nil {
			defer req.Body.Close()
			body, _ := io.ReadAll(req.Body)

			if len(body) > 0 {
				app.Logger.Infof("body: %s", string(body))
			}
		}

		m := &model.Mission{
			Name:           getStringParam(req, "missionname"),
			Username:       username,
			CreatorUID:     getStringParam(req, "creatorUid"),
			CreateTime:     time.Now(),
			LastEdit:       time.Now(),
			BaseLayer:      getStringParam(req, "baseLayer"),
			Bbox:           getStringParam(req, "bbox"),
			ChatRoom:       getStringParam(req, "chatRoom"),
			Classification: getStringParam(req, "classification"),
			Description:    getStringParam(req, "description"),
			InviteOnly:     getBoolParam(req, "inviteOnly", false),
			Password:       getStringParam(req, "password"),
			Path:           getStringParam(req, "path"),
			Tool:           getStringParam(req, "tool"),
			Groups:         strings.Join(getStringParams(req, "group"), ","),
			Keywords:       "",
		}

		if err := app.missions.PutMission(m); err != nil {
			res.Status = http.StatusBadRequest
			return res.WriteString(err.Error())
		}

		return res.WriteJSON(makeAnswer("Mission", []*model.MissionDTO{model.ToMissionDTO(m)}))
	}
}

func getMissionDeleteHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		mname := getStringParam(req, "missionname")
		m := app.missions.GetMission(mname)

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		app.missions.DeleteMission(mname)
		return nil
	}
}

func getMissionsInvitationsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		uid := getStringParam(req, "clientUid")

		return res.WriteJSON(makeAnswer("MissionInvitation", app.missions.GetInvitations(uid)))
	}
}

func getMissionRoleHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		mname := getStringParam(req, "missionname")
		m := app.missions.GetMission(mname)

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		return res.WriteJSON(makeAnswer("MissionRole", model.GetRole("")))
	}
}

func getMissionRolePutHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		mname := getStringParam(req, "missionname")
		m := app.missions.GetMission(mname)

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		printParams(req, app.Logger)

		if req.Body != nil {
			defer req.Body.Close()
			body, _ := io.ReadAll(req.Body)

			if len(body) > 0 {
				app.Logger.Infof("body: %s", string(body))
			}
		}

		return res.WriteJSON(makeAnswer("MissionRole", model.GetRole("")))
	}
}

func getMissionLogHandler(app *App) func(req *air.Request, res *air.Response) error {
	result := makeAnswer("com.bbn.marti.sync.model.LogEntry", []*model.MissionLogEntryDTO{})

	return func(req *air.Request, res *air.Response) error {
		m := app.missions.GetMission(getStringParam(req, "missionname"))

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		return res.WriteJSON(result)
	}
}

func getMissionKeywordsPutHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		m := app.missions.GetMission(getStringParam(req, "missionname"))

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		if req.Body == nil {
			return nil
		}

		defer req.Body.Close()
		b, err := io.ReadAll(req.Body)

		if err != nil {
			return err
		}

		var kw []string

		if err := json.Unmarshal(b, &kw); err != nil {
			return err
		}

		app.missions.AddKw(getStringParam(req, "missionname"), kw)

		return nil
	}
}

func getMissionSubscriptionsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		missionName := getStringParam(req, "missionname")
		m := app.missions.GetMission(missionName)

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		return res.WriteJSON(makeAnswer("MissionSubscription", app.missions.GetSubscribers(missionName)))
	}
}

func getMissionSubscriptionHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		mname := getStringParam(req, "missionname")
		m := app.missions.GetMission(mname)

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		s := app.missions.GetSubscription(mname, getStringParam(req, "uid"))
		if s == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		return res.WriteJSON(makeAnswer("MissionSubscription", model.ToMissionSubscriptionDTO(s)))
	}
}

func getMissionSubscriptionPutHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		m := app.missions.GetMission(getStringParam(req, "missionname"))

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		printParams(req, app.Logger)

		if m.InviteOnly {
			res.Status = http.StatusForbidden
			return res.WriteString("Illegal attempt to subscribe to invite only mission!")
		}

		if m.Password != "" && getStringParam(req, "password") != m.Password {
			res.Status = http.StatusForbidden
			return res.WriteString("Illegal attempt to subscribe to mission! Password did not match.")
		}

		s := &model.Subscription{
			MissionName: getStringParam(req, "missionname"),
			ClientUID:   getStringParam(req, "uid"),
			Username:    getUsernameFromReq(req),
			CreateTime:  time.Now(),
			Role:        "",
		}

		app.missions.PutSubscription(s)

		return res.WriteJSON(makeAnswer("MissionSubscription", model.ToMissionSubscriptionDTO(s)))
	}
}

func getMissionSubscriptionDeleteHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		m := app.missions.GetMission(getStringParam(req, "missionname"))

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		app.missions.DeleteSubscription(getStringParam(req, "missionname"), getStringParam(req, "uid"))

		return nil
	}
}

func getMissionSubscriptionRolesHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		missionName := getStringParam(req, "missionname")
		m := app.missions.GetMission(missionName)

		if m == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		s := app.missions.GetSubscriptions(missionName)

		return res.WriteJSON(makeAnswer("MissionSubscription", model.ToMissionSubscriptionsDTO(s)))
	}
}

func getMissionChangesHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		name := getStringParam(req, "missionname")
		d1 := time.Now().Add(-time.Second * time.Duration(getIntParam(req, "secago", 31536000)))

		mission := app.missions.GetMission(name)

		if mission == nil {
			res.Status = http.StatusNotFound
			return nil
		}

		result := make([]*model.MissionChangeDTO, 0)

		for _, item := range mission.Items {
			if item.Timestamp.After(d1) {
				result = append(result, model.NewAddChangeItem(name, item))
			}
		}

		if mission.CreateTime.After(d1) {
			result = append(result, model.NewCreateChange(mission))
		}

		return res.WriteJSON(makeAnswer("MissionChange", result))
	}
}

func getMissionCotHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		name := getStringParam(req, "missionname")
		mission := app.missions.GetMission(name)
		if mission == nil {
			res.Status = http.StatusNotFound

			return nil
		}

		res.Header.Set("Content-Type", "application/xml")

		fb := new(strings.Builder)
		fb.WriteString("<?xml version='1.0' encoding='UTF-8' standalone='yes'?>\n")
		fb.WriteString("<events>\n")
		enc := xml.NewEncoder(fb)

		for _, item := range mission.Items {
			if err := enc.Encode(cot.CotToEvent(item.Event)); err != nil {
				app.Logger.Errorf("xml encode error %v", err)
			}
		}

		fb.WriteString("\n</events>")

		return res.WriteString(fb.String())
	}
}

func getMissionContactsHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		name := getStringParam(req, "missionname")
		if app.missions.GetMission(name) == nil {
			res.Status = http.StatusNotFound

			return nil
		}

		return res.WriteJSON([]any{})
	}
}

func getMissionContentDeleteHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		name := getStringParam(req, "missionname")
		mission := app.missions.GetMission(name)

		if mission == nil {
			res.Status = http.StatusNotFound

			return nil
		}

		app.missions.DeleteMissionPoints(mission.ID, getStringParam(req, "uid"))

		return nil
	}
}

func getInvitePutHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		name := getStringParam(req, "missionname")
		if app.missions.GetMission(name) == nil {
			res.Status = http.StatusNotFound

			return nil
		}

		// type can be: clientUid, callsign, userName, group, team
		typ := getStringParam(req, "type")

		if typ != "clientUid" {
			app.Logger.Warnf("we do not support invitation with type %s now", typ)
			res.Status = http.StatusBadRequest
			return nil
		}

		printParams(req, app.Logger)

		if req.Body != nil {
			defer req.Body.Close()
			body, _ := io.ReadAll(req.Body)

			if len(body) > 0 {
				app.Logger.Infof("body: %s", string(body))
			}
		}

		inv := &model.Invitation{
			MissionName: name,
			Typ:         typ,
			Invitee:     getStringParam(req, "uid"),
			CreatorUID:  getStringParam(req, "creatorUid"),
			CreateTime:  time.Now(),
			Role:        getStringParam(req, "role"),
		}

		app.missions.PutInvitation(inv)

		return nil
	}
}

func getInviteDeleteHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		name := getStringParam(req, "missionname")
		if app.missions.GetMission(name) == nil {
			res.Status = http.StatusNotFound

			return nil
		}

		app.missions.DeleteInvitation(name, getStringParam(req, "uid"), getStringParam(req, "type"))
		return nil
	}
}

func printParams(req *air.Request, logger *zap.SugaredLogger) {
	params := []string{}
	for _, r := range req.Params() {
		params = append(params, r.Name+"="+r.Value().String())
	}

	logger.Infof("params: %s", strings.Join(params, ","))
}
