package model

import (
	"strings"
	"time"

	"github.com/kdudkov/goatak/pkg/cot"
)

type CotTime time.Time

func (s CotTime) MarshalJSON() ([]byte, error) {
	t := time.Time(s)
	return []byte(t.UTC().Format("\"2006-01-02T15:04:05.999Z07:00\"")), nil
}

type MissionDTO struct {
	Name              string            `json:"name"`
	CreatorUID        string            `json:"creatorUid"`
	CreateTime        CotTime           `json:"createTime"`
	LastEdit          CotTime           `json:"lastEdited"`
	BaseLayer         string            `json:"baseLayer"`
	Bbox              string            `json:"bbox"`
	ChatRoom          string            `json:"chatRoom"`
	Classification    string            `json:"classification"`
	Contents          []*ContentItemDTO `json:"contents"`
	DefaultRole       *MissionRoleDTO   `json:"defaultRole,omitempty"`
	OwnerRole         *MissionRoleDTO   `json:"ownerRole,omitempty"`
	Description       string            `json:"description"`
	Expiration        int               `json:"expiration"`
	ExternalData      []any             `json:"externalData"`
	Feeds             []string          `json:"feeds"`
	Groups            []string          `json:"groups"`
	InviteOnly        bool              `json:"inviteOnly"`
	Keywords          []string          `json:"keywords"`
	MapLayers         []string          `json:"mapLayers"`
	PasswordProtected bool              `json:"passwordProtected"`
	Path              string            `json:"path"`
	Tool              string            `json:"tool"`
	Uids              []*MissionItemDTO `json:"uids"`
}

type MissionRoleDTO struct {
	Type        string   `json:"type"`
	Permissions []string `json:"permissions"`
}

type ContentItemDTO struct {
	CreatorUID string    `json:"creatorUid"`
	Timestamp  time.Time `json:"timestamp"`
	Data       DataDTO   `json:"data"`
}

type DataDTO struct {
	UID            string   `json:"uid"`
	Name           string   `json:"name"`
	Keywords       []string `json:"keywords"`
	MimeType       string   `json:"mimeType"`
	SubmissionTime CotTime  `json:"submissionTime"`
	Submitter      string   `json:"submitter"`
	CreatorUID     string   `json:"creatorUid"`
	Hash           string   `json:"hash"`
	Size           int      `json:"size"`
}

type MissionItemDTO struct {
	CreatorUID string             `json:"creatorUid"`
	Timestamp  CotTime            `json:"timestamp"`
	Data       string             `json:"data"`
	Details    *MissionDetailsDTO `json:"details"`
}

type MissionSubscriptionDTO struct {
	ClientUID  string          `json:"clientUid"`
	Username   string          `json:"username"`
	CreateTime CotTime         `json:"createTime"`
	Role       *MissionRoleDTO `json:"role"`
}

type MissionChange struct {
	Type        string             `json:"type"`
	MissionName string             `json:"missionName"`
	Timestamp   CotTime            `json:"timestamp"`
	CreatorUID  string             `json:"creatorUid"`
	ServerTime  CotTime            `json:"serverTime"`
	ContentUID  string             `json:"contentUid,omitempty"`
	Details     *MissionDetailsDTO `json:"details,omitempty"`
}

type MissionDetailsDTO struct {
	Type        string       `json:"type"`
	Callsign    string       `json:"callsign"`
	Title       string       `json:"title,omitempty"`
	IconsetPath string       `json:"iconsetPath"`
	Color       string       `json:"color"`
	Location    *LocationDTO `json:"location,omitempty"`
}

type LocationDTO struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type MissionLogEntryDTO struct {
	Content       string    `json:"content"`
	ContentHashes []string  `json:"contentHashes"`
	Created       time.Time `json:"created"`
	CreatorUID    string    `json:"creatorUid"`
	Dtg           time.Time `json:"dtg"`
	ID            string    `json:"id"`
	Keywords      []string  `json:"keywords"`
	MissionNames  []string  `json:"missionNames"`
	Servertime    time.Time `json:"servertime"`
	EntryUID      string    `json:"entryUid"`
}

func ToMissionDTO(m *Mission) *MissionDTO {
	if m == nil {
		return nil
	}

	return &MissionDTO{
		Name:           m.Name,
		CreatorUID:     m.CreatorUID,
		CreateTime:     CotTime(m.CreateTime),
		LastEdit:       CotTime(m.LastEdit),
		BaseLayer:      m.BaseLayer,
		Bbox:           m.Bbox,
		ChatRoom:       m.ChatRoom,
		Classification: m.Classification,
		Contents:       nil,
		DefaultRole:    NewRole("MISSION_SUBSCRIBER", "MISSION_WRITE", "MISSION_READ"),
		OwnerRole: NewRole("MISSION_OWNER", "MISSION_MANAGE_FEEDS", "MISSION_SET_PASSWORD",
			"MISSION_WRITE", "MISSION_MANAGE_LAYERS", "MISSION_UPDATE_GROUPS", "MISSION_READ", "MISSION_DELETE",
			"MISSION_SET_ROLE"),
		Description:       m.Description,
		Expiration:        0,
		ExternalData:      nil,
		Feeds:             nil,
		Groups:            strings.Split(m.Groups, ","),
		InviteOnly:        m.InviteOnly,
		Keywords:          strings.Split(m.Keywords, ","),
		MapLayers:         nil,
		PasswordProtected: m.Password != "",
		Path:              m.Path,
		Tool:              m.Tool,
		Uids:              nil,
	}
}

func ToMissionSubscriptionDTO(s *Subscription) *MissionSubscriptionDTO {
	if s == nil {
		return nil
	}

	return &MissionSubscriptionDTO{
		ClientUID:  s.ClientUID,
		Username:   s.Username,
		CreateTime: CotTime(s.CreateTime),
		Role:       NewRole(s.RoleType, strings.Split(s.Permissions, ",")...),
	}
}

func ToMissionSubscriptionsDTO(subscriptions []*Subscription) []*MissionSubscriptionDTO {
	res := make([]*MissionSubscriptionDTO, len(subscriptions))

	for i, s := range subscriptions {
		res[i] = ToMissionSubscriptionDTO(s)
	}

	return res
}

func NewCreateChange(m *MissionDTO) *MissionChange {
	return &MissionChange{
		Type:        "CREATE_MISSION",
		MissionName: m.Name,
		CreatorUID:  m.CreatorUID,
		Timestamp:   CotTime(time.Now()),
		ServerTime:  CotTime(time.Now()),
	}
}

func NewDetails(msg *cot.CotMessage) *MissionDetailsDTO {
	return &MissionDetailsDTO{
		Type:        msg.GetType(),
		Callsign:    msg.GetCallsign(),
		IconsetPath: msg.GetIconsetPath(),
		Color:       msg.GetColor(),
		Location: &LocationDTO{
			Lat: msg.GetLat(),
			Lon: msg.GetLon(),
		},
	}
}

func NewAddChange(name string, msg *cot.CotMessage) *MissionChange {
	creator, _ := msg.GetParent()

	return &MissionChange{
		Type:        "ADD_CONTENT",
		MissionName: name,
		CreatorUID:  creator,
		Timestamp:   CotTime(time.Now()),
		ServerTime:  CotTime(time.Now()),
		Details:     NewDetails(msg),
	}
}

func NewUID(msg *cot.CotMessage) *MissionItemDTO {
	creator, _ := msg.GetParent()

	return &MissionItemDTO{
		CreatorUID: creator,
		Timestamp:  CotTime(time.Now()),
		Data:       msg.GetUID(),
		Details:    NewDetails(msg),
	}
}

func NewRole(typ string, perms ...string) *MissionRoleDTO {
	return &MissionRoleDTO{
		Type:        typ,
		Permissions: perms,
	}
}