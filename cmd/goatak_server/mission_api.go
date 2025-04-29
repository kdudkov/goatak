package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
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
	g.Put("/:missionname/contents/missionpackage", getMissionContentPackagePutHandler(app))
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
		user := app.users.Get(Username(ctx))

		data := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).Full().Get()
		result := make([]*model.MissionDTO, len(data))

		app.logger.Info(fmt.Sprintf("got %d missions for scope %s", len(data), user.GetScope()))
		for i, m := range data {
			result[i] = model.ToMissionDTO(m, false)
		}

		return ctx.JSON(makeAnswer(missionType, result))
	}
}

func getMissionHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).Full().One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(m, false)}))
	}
}

func getMissionPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		user := app.users.Get(username)

		body := ctx.Body()

		if len(body) > 0 {
			app.logger.Info("body: " + string(body))
		}

		m := &model.Mission{
			Name:           ctx.Params("missionname"),
			Scope:          user.GetScope(),
			Creator:        username,
			CreatorUID:     ctx.Query("creatorUid"),
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

		if err := app.dbm.CreateMission(m); err != nil {
			app.logger.Warn("mission add error", slog.Any("error", err))
			return ctx.Status(fiber.StatusConflict).SendString(err.Error())
		}

		if !m.InviteOnly {
			app.NewCotMessage(model.MissionCreateNotificationMsg(m))
		}

		return ctx.Status(fiber.StatusCreated).
			JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(m, true)}))
	}
}

func getMissionDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).Full().One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		app.dbm.MissionQuery().Delete(m.ID)

		return ctx.JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(m, false)}))
	}
}

func getMissionsInvitationsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Query("clientUid")

		return ctx.JSON(makeAnswer(missionInvitationType, app.dbm.GetInvitations(uid)))
	}
}

func getMissionRoleHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionRoleType, model.GetRole("")))
	}
}

func getMissionRolePutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).Full().One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionRoleType, model.GetRole("")))
	}
}

func getMissionLogHandler(app *App) fiber.Handler {
	result := makeAnswer(logEntryType, []*model.MissionLogEntryDTO{})

	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(result)
	}
}

func getMissionKeywordsPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		var kw []string

		if err := json.Unmarshal(ctx.Body(), &kw); err != nil {
			return err
		}

		return app.dbm.UpdateKw(ctx.Params("missionname"), user.GetScope(), kw)
	}
}

func getMissionSubscriptionsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionSubscriptionType, app.dbm.GetSubscribers(m.ID)))
	}
}

func getMissionSubscriptionHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		s := app.dbm.SubscriptionQuery().Mission(m.ID).Client(ctx.Query("uid")).One()
		if s == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON(makeAnswer(missionSubscriptionType, model.ToMissionSubscriptionDTO(s, m.Token)))
	}
}

func getMissionSubscriptionPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))

		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		s, err := app.dbm.Subscribe(user, m, ctx.Query("uid"), ctx.Query("password"))

		if err != nil {
			return ctx.Status(fiber.StatusForbidden).SendString(err.Error())
		}

		return ctx.Status(fiber.StatusCreated).JSON(
			makeAnswer(missionSubscriptionType, model.ToMissionSubscriptionDTO(s, m.Token)),
		)
	}
}

func getMissionSubscriptionDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).Name(ctx.Params("missionname")).One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		app.dbm.SubscriptionQuery().Mission(m.ID).Client(ctx.Query("uid")).Delete()

		return nil
	}
}

func getMissionSubscriptionRolesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		s := app.dbm.SubscriptionQuery().Mission(m.ID).Get()

		return ctx.JSON(makeAnswer(missionSubscriptionType, model.ToMissionSubscriptionsDTO(s)))
	}
}

func getMissionChangesHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		mission := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()
		d1 := time.Now().Add(-time.Second * time.Duration(ctx.QueryInt("secago", 31536000)))

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		ch := app.dbm.GetChanges(mission.ID, d1, ctx.QueryBool("squashed"))

		result := make([]*model.MissionChangeDTO, len(ch))

		for i, c := range ch {
			result[i] = model.ToChangeDTO(c, mission.Name)
		}

		return ctx.JSON(makeAnswer(missionChangeType, result))
	}
}

func getMissionCotHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		mission := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).Full().One()

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		ctx.Set(fiber.HeaderContentType, "application/xml")

		fb := new(strings.Builder)
		fb.WriteString("<?xml version='1.0' encoding='UTF-8' standalone='yes'?>\n")
		fb.WriteString("<events>\n")
		enc := xml.NewEncoder(fb)

		for _, item := range mission.Points {
			if err := enc.Encode(cot.CotToEvent(item.GetEvent())); err != nil {
				app.logger.Error("xml encode error", slog.Any("error", err))
			}
		}

		fb.WriteString("\n</events>")

		return ctx.SendString(fb.String())
	}
}

func getMissionContactsHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		m := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).Full().One()

		if m == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		return ctx.JSON([]any{})
	}
}

func getMissionContentPutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		mission := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()
		author := ctx.Query("creatorUid")

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		var data map[string][]string

		if err := json.Unmarshal(ctx.Body(), &data); err != nil {
			return ctx.SendStatus(fiber.StatusInternalServerError)
		}

		var added = false

		if d, ok := data["hashes"]; ok {
			for _, h := range d {
				change := app.dbm.AddMissionResource(mission, h, author)

				app.notifyMissionSubscribers(mission, change)
			}
		}

		if added {
			ctx.Status(fiber.StatusCreated)
		}

		return ctx.JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(mission, false)}))
	}
}

func getMissionContentPackagePutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		mission := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()
		_ = ctx.Query("creatorUid")

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		f, err := os.CreateTemp("", "tak_pkg_*.zip")

		if err != nil {
			app.logger.Error("error open temp file", slog.Any("error", err))
			return err
		}

		defer func() {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}()

		n, err := io.Copy(f, ctx.Request().BodyStream())

		if err != nil {
			app.logger.Error("error", slog.Any("error", err))
			return err
		}

		_ = f.Close()

		app.logger.Info(fmt.Sprintf("got package file with size %d", n))
		z, err := zip.OpenReader(f.Name())

		if err != nil {
			app.logger.Error("error", slog.Any("error", err))
			return err
		}

		defer z.Close()

		for _, zf := range z.File {
			app.logger.Info("file " + zf.Name)

			if zf.Name == "changes.json" {
				f1, _ := zf.Open()
				content, _ := io.ReadAll(f1)
				f1.Close()
				fmt.Println(content)
			}
		}

		d1 := time.Now().Add(-time.Second * time.Duration(ctx.QueryInt("secago", 31536000)))
		ch := app.dbm.GetChanges(mission.ID, d1, ctx.QueryBool("squashed"))
		result := make([]*model.MissionChangeDTO, len(ch))

		for i, c := range ch {
			result[i] = model.ToChangeDTO(c, mission.Name)
		}

		return ctx.JSON(makeAnswer(missionChangeType, result))
	}
}

func getMissionContentDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		mission := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).Full().One()

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		author := ctx.Query("creatorUid")

		if uid := ctx.Query("uid"); uid != "" {
			change := app.dbm.DeleteMissionPoint(mission, uid, author)

			app.notifyMissionSubscribers(mission, change)
		}

		if hash := ctx.Query("hash"); hash != "" {
			change := app.dbm.DeleteMissionContent(mission, hash, author)

			app.notifyMissionSubscribers(mission, change)
		}

		m1 := app.dbm.MissionQuery().Id(mission.ID).Full().One()

		return ctx.JSON(makeAnswer(missionType, []*model.MissionDTO{model.ToMissionDTO(m1, false)}))
	}
}

func getInvitePutHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		mission := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()

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
			Role:       ctx.Query("role"),
		}

		_, err := app.dbm.Invite(inv)

		return err
	}
}

func getInviteDeleteHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := app.users.Get(Username(ctx))
		mission := app.dbm.MissionQuery().Scope(user.GetScope()).ReadScope(user.GetReadScope()).
			Name(ctx.Params("missionname")).One()

		if mission == nil {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		app.dbm.InvitationQuery().Mission(mission.ID).Invitee(ctx.Params("uid")).
			Type(ctx.Params("type")).Delete()

		return nil
	}
}
