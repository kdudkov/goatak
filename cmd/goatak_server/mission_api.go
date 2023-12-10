package main

import (
	"github.com/aofei/air"

	"github.com/kdudkov/goatak/internal/model"
)

func addMissionApi(app *App, api *air.Air) {
	g := api.Group("/Marti/api/missions")

	g.GET("", getMissionsHandler(app))
	g.GET("/all/invitations", getMissionsInvitationsHandler(app))
	g.GET("/:missionname", getMissionHandler(app))
}

func getMissionsHandler(app *App) func(req *air.Request, res *air.Response) error {
	result := makeAnswer("Mission", []string{})

	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON(result)
	}
}

func getMissionsInvitationsHandler(app *App) func(req *air.Request, res *air.Response) error {
	result := makeAnswer("MissionInvitation", []string{})

	return func(req *air.Request, res *air.Response) error {
		return res.WriteJSON(result)
	}
}

func getMissionHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		m := model.GetDefaultMission(getStringParam(req, "missionname"))

		return res.WriteJSON(makeAnswer("Mission", []any{m}))
	}
}
