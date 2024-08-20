package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
)

const (
	missionSubscriptionType = "com.bbn.marti.sync.model.MissionSubscription"
	missionType             = "Mission"
	missionInvitationType   = "MissionInvitation"
	missionRoleType         = "MissionRole"
	logEntryType            = "com.bbn.marti.sync.model.LogEntry"
	missionChangeType       = "MissionChange"
)

func addMissionApi(app *App, f fiber.Router) {
	g := f.Group("/Marti/api/missions")

	g.Get("/", getMissionsHandler(app))

	g.Get("/all/invitations", getMissionsInvitationsHandler(app))

	g.Get("/:missionname", getMissionHandler(app))
	g.Put("/:missionname", getMissionPutHandler(app))
	g.Delete("/:missionname", getMissionDeleteHandler(app))
	g.Get("/:missionname/changes", getMissionChangesHandler(app))
	g.Get("/:missionname/cot", getMissionCotHandler(app))
	g.Get("/:missionname/contacts", getMissionContactsHandler(app))
	g.Put("/:missionname/contents", getMissionContentPutHandler(app))
	g.Delete("/:missionname/contents", getMissionContentDeleteHandler(app))
	g.Get("/:missionname/log", getMissionLogHandler(app))
	g.Put("/:missionname/keywords", getMissionKeywordsPutHandler(app))
	g.Get("/:missionname/role", getMissionRoleHandler(app))
	g.Put("/:missionname/role", getMissionRolePutHandler(app))
	g.Get("/:missionname/subscription", getMissionSubscriptionHandler(app))
	g.Put("/:missionname/subscription", getMissionSubscriptionPutHandler(app))
	g.Delete("/:missionname/subscription", getMissionSubscriptionDeleteHandler(app))
	g.Get("/:missionname/subscriptions", getMissionSubscriptionsHandler(app))
	g.Get("/:missionname/subscriptions/roles", getMissionSubscriptionRolesHandler(app))
	g.Put("/:missionname/invite/:type/:uid", getInvitePutHandler(app))
	g.Delete("/:missionname/invite/:type/:uid", getInviteDeleteHandler(app))
}

func getMissionsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))

		data := app.missions.GetAllMissions(user.GetScope())
		result := make([]*model.MissionDTO, len(data))

		for i, m := range data {
			result[i] = model.ToMissionDTO(m, app.packageManager, false)
		}

		return ctx.JSON(makeAnswer(missionType, result))
	}
}

func getMissionHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(m, app.packageManager, false)}))
	}
}

func getMissionPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		user := app.users.GetUser(username)

		body := ctx.Body()

		if len(body) > 0 {
			app.logger.Info("body: " + string(body))
		}

		m := &model.Mission{
			Name:           ctx.Params("missionname"),
			Scope:          user.GetScope(),
			Username:       username,
			CreatorUID:     ctx.Query("creatorUid"),
			CreateTime:     time.Now(),
			LastEdit:       time.Now(),
			BaseLayer:      ctx.Query("baseLayer"),
			Bbox:           ctx.Query("bbox"),
			ChatRoom:       ctx.Query("chatRoom"),
			Classification: ctx.Query("classification"),
			Description:    ctx.Query("description"),
			InviteOnly:     ctx.QueryBool("inviteOnly", false),
			Password:       ctx.Query("password"),
			Path:           ctx.Query("path"),
			Tool:           ctx.Query("tool"),
			Groups:         "",
			Keywords:       "",
			Token:          uuid.NewString(),
		}

		if err := app.missions.PutMission(m); err != nil {
			app.logger.Warn("mission add error", "error", err.Error())
			return ctx.Status(fiber.StatusConflict).SendString(err.Error())
		}

		if !m.InviteOnly {
			app.NewCotMessage(model.MissionCreateNotificationMsg(m))
		}

		return ctx.Status(fiber.StatusCreated).
			JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(m, app.packageManager, true)}))
	}
}

func getMissionDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		app.missions.DeleteMission(m.ID)

		return ctx.JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(m, app.packageManager, false)}))
	}
}

func getMissionsInvitationsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Query("clientUid")

		return ctx.JSON(makeAnswer(missionInvitationType, app.missions.GetInvitations(uid)))
	}
}

func getMissionRoleHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionRoleType, model.GetRole("")))
	}
}

func getMissionRolePutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionRoleType, model.GetRole("")))
	}
}

func getMissionLogHandler(app *App) fiber.Handler {
	result := makeAnswer(logEntryType, []*model.MissionLogEntryDTO{})

	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(result)
	}
}

func getMissionKeywordsPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		var kw []string

		if err := json.Unmarshal(ctx.Body(), &kw); err != nil {
			return err
		}

		app.missions.AddKw(ctx.Params("missionname"), kw)

		return nil
	}
}

func getMissionSubscriptionsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionSubscriptionType, app.missions.GetSubscribers(m.ID)))
	}
}

func getMissionSubscriptionHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		s := app.missions.GetSubscription(m.ID, ctx.Query("uid"))
		if s == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionSubscriptionType, model.ToMissionSubscriptionDTO(s, m.Token)))
	}
}

func getMissionSubscriptionPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		if m.InviteOnly {
			return ctx.Status(fiber.StatusForbidden).SendString("Illegal attempt to subscribe to invite only mission!")
		}

		if m.Password != "" && ctx.Query("password") != m.Password {
			return ctx.Status(fiber.StatusForbidden).SendString("Illegal attempt to subscribe to mission! Password did not match.")
		}

		s := &model.Subscription{
			MissionID:  m.ID,
			ClientUID:  ctx.Query("uid"),
			Username:   Username(ctx),
			CreateTime: time.Now(),
			Role:       "MISSION_SUBSCRIBER",
		}

		app.missions.PutSubscription(s)

		return ctx.Status(fiber.StatusCreated).JSON(makeAnswer(missionSubscriptionType, model.ToMissionSubscriptionDTO(s, m.Token)))
	}
}

func getMissionSubscriptionDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		app.missions.DeleteSubscription(ctx.Params("missionname"), ctx.Query("uid"))

		return nil
	}
}

func getMissionSubscriptionRolesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		s := app.missions.GetSubscriptions(m.ID)

		return ctx.JSON(makeAnswer(missionSubscriptionType, model.ToMissionSubscriptionsDTO(s)))
	}
}

func getMissionChangesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		mission := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))
		d1 := time.Now().Add(-time.Second * time.Duration(ctx.QueryInt("secago", 31536000)))

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		ch := app.missions.GetChanges(mission.ID, d1)

		result := make([]*model.MissionChangeDTO, len(ch))

		for i, c := range ch {
			result[i] = model.NewChangeDTO(c, mission.Name)
		}

		return ctx.JSON(makeAnswer(missionChangeType, result))
	}
}

func getMissionCotHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		mission := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		ctx.Set(fiber.HeaderContentType, "application/xml")

		fb := new(strings.Builder)
		fb.WriteString("<?xml version='1.0' encoding='UTF-8' standalone='yes'?>\n")
		fb.WriteString("<events>\n")
		enc := xml.NewEncoder(fb)

		for _, item := range mission.Items {
			if err := enc.Encode(cot.CotToEvent(item.GetEvent())); err != nil {
				app.logger.Error("xml encode error", "error", err)
			}
		}

		fb.WriteString("\n</events>")

		return ctx.SendString(fb.String())
	}
}

func getMissionContactsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		m := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON([]any{})
	}
}

func getMissionContentPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		mission := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		var data map[string][]string

		if err := json.Unmarshal(ctx.Body(), &data); err != nil {
			return ctx.SendStatus(fiber.StatusInternalServerError)
		}

		var added = false

		if d, ok := data["hashes"]; ok {
			added = mission.AddHashes(d...)
		}

		if added {
			mission.LastEdit = time.Now()
			app.missions.Save(mission)

			ctx.Status(fiber.StatusCreated)
		}

		return ctx.JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(mission, app.packageManager, false)}))
	}
}

func getMissionContentDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		mission := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		author := ctx.Query("creatorUid")

		if uid := ctx.Query("uid"); uid != "" {
			change := app.missions.DeleteMissionPoint(mission.ID, uid, author)

			app.notifyMissionSubscribers(mission, change)
		}

		if hash := ctx.Query("hash"); hash != "" {
			app.missions.DeleteMissionContent(mission.ID, hash, author)
		}

		m1 := app.missions.GetMissionById(mission.ID)

		return ctx.JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(m1, app.packageManager, false)}))
	}
}

func getInvitePutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		mission := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		// type can be: clientUid, callsign, userName, group, team
		typ := ctx.Params("type")

		if typ != "clientUid" {
			app.logger.Warn(fmt.Sprintf("we do not support invitation with type %s now", typ))
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		inv := &model.Invitation{
			MissionID:  mission.ID,
			Typ:        typ,
			Invitee:    ctx.Params("uid"),
			CreatorUID: ctx.Query("creatorUid"),
			CreateTime: time.Now(),
			Role:       ctx.Query("role"),
		}

		app.missions.PutInvitation(inv)

		return nil
	}
}

func getInviteDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.GetUser(Username(ctx))
		mission := app.missions.GetMission(user.GetScope(), ctx.Params("missionname"))

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		app.missions.DeleteInvitation(mission.ID, ctx.Params("uid"), ctx.Params("type"))

		return nil
	}
}
