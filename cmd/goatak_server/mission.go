package main

import "time"

type Mission struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	ChatRoom       string    `json:"chatRoom"`
	BaseLayer      string    `json:"baseLayer"`
	Bbox           string    `json:"bbox"`
	Path           string    `json:"path"`
	Classification string    `json:"classification"`
	Tool           string    `json:"tool"`
	Keywords       []string  `json:"keywords"`
	CreatorUID     string    `json:"creatorUid"`
	CreateTime     time.Time `json:"createTime"`
	ExternalData   []any     `json:"externalData"`
	Feeds          []any     `json:"feeds"`
	MapLayers      []any     `json:"mapLayers"`
	DefaultRole    struct {
		Permissions []string `json:"permissions"`
		Type        string   `json:"type"`
	} `json:"defaultRole"`
	OwnerRole struct {
		Permissions []string `json:"permissions"`
		Type        string   `json:"type"`
	} `json:"ownerRole"`
	InviteOnly        bool     `json:"inviteOnly"`
	Expiration        int      `json:"expiration"`
	GUID              string   `json:"guid"`
	Uids              []string `json:"uids"`
	Contents          []any    `json:"contents"`
	Token             string   `json:"token"`
	PasswordProtected bool     `json:"passwordProtected"`
}

func GetDefault(name string) *Mission {
	m := new(Mission)

	m.Name = name
	m.DefaultRole.Type = "MISSION_SUBSCRIBER"
	m.DefaultRole.Permissions = []string{"MISSION_WRITE", "MISSION_READ"}
	m.OwnerRole.Type = "MISSION_OWNER"
	m.OwnerRole.Permissions = []string{"MISSION_MANAGE_FEEDS", "MISSION_SET_PASSWORD", "MISSION_WRITE", "MISSION_MANAGE_LAYERS", "MISSION_UPDATE_GROUPS", "MISSION_READ", "MISSION_DELETE", "MISSION_SET_ROLE"}

	return m
}
