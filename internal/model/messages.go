package model

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

const missionNotificationStale = time.Second * 5

func MissionChangeNotificationMsg(missionName string, scope string, c *Change) *cot.CotMessage {
	msg := cot.BasicMsg("t-x-m-c", uuid.NewString(), missionNotificationStale)
	msg.CotEvent.How = "h-g-i-g-o"

	xd := cot.NewXMLDetails()

	ch := xd.AddChild("mission", map[string]string{"type": "CHANGE", "name": missionName}, "").
		AddChild("MissionChanges", nil, "").AddChild("MissionChange", nil, "")

	ch.AddChild("contentUid", nil, c.ContentUID)
	ch.AddChild("type", nil, c.Type)
	ch.AddChild("isFederatedChange", nil, "false")
	ch.AddChild("missionName", nil, missionName)
	ch.AddChild("timestamp", nil, strconv.Itoa(int(c.CreateTime.Unix())))

	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}

	return &cot.CotMessage{From: "local", TakMessage: msg, Detail: xd, Scope: scope}
}

func MissionCreateNotificationMsg(m *Mission) *cot.CotMessage {
	msg := cot.BasicMsg("t-x-m-n", uuid.NewString(), missionNotificationStale)
	msg.CotEvent.How = "h-g-i-g-o"

	xd := cot.NewXMLDetails()

	params := map[string]string{"type": "CREATE", "name": m.Name, "creatorUid": m.CreatorUID}

	if m.Tool != "" {
		params["tool"] = m.Tool
	}

	xd.AddChild("mission", params, "")

	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}

	return &cot.CotMessage{From: "local", TakMessage: msg, Detail: xd, Scope: m.Scope}
}
