package model

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

func MissionChangePountMsg(missionName string, c *Change) *cotproto.TakMessage {
	msg := cot.BasicMsg("t-x-m-c", uuid.NewString(), time.Second*5)
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

	return msg
}
