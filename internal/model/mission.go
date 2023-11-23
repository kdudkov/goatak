package model

import "time"

type Mission struct {
	Name           string         `json:"name"`
	CreatorUID     string         `json:"creatorUid"`
	CreateTime     time.Time      `json:"createTime"`
	BaseLayer      string         `json:"baseLayer"`
	Bbox           string         `json:"bbox"`
	ChatRoom       string         `json:"chatRoom"`
	Classification string         `json:"classification"`
	Contents       []*ContentItem `json:"contents"`
	DefaultRole    struct {
		Permissions []string `json:"permissions"`
		Type        string   `json:"type"`
	} `json:"defaultRole"`
	OwnerRole struct {
		Permissions []string `json:"permissions"`
		Type        string   `json:"type"`
	} `json:"ownerRole"`
	Description       string         `json:"description"`
	Expiration        int            `json:"expiration"`
	ExternalData      []any          `json:"externalData"`
	Feeds             []any          `json:"feeds"`
	Groups            []any          `json:"groups"`
	InviteOnly        bool           `json:"inviteOnly"`
	Keywords          []string       `json:"keywords"`
	MapLayers         []any          `json:"mapLayers"`
	PasswordProtected bool           `json:"passwordProtected"`
	Path              string         `json:"path"`
	Tool              string         `json:"tool"`
	Uids              []*MissionItem `json:"uids"`
}

type ContentItem struct {
	CreatorUID string    `json:"creatorUid"`
	Timestamp  time.Time `json:"timestamp"`
	Data       struct {
		UID            string    `json:"uid"`
		Name           string    `json:"name"`
		Keywords       []string  `json:"keywords"`
		MimeType       string    `json:"mimeType"`
		SubmissionTime time.Time `json:"submissionTime"`
		Submitter      string    `json:"submitter"`
		CreatorUID     string    `json:"creatorUid"`
		Hash           string    `json:"hash"`
		Size           int       `json:"size"`
	} `json:"data"`
}

type MissionItem struct {
	CreatorUID string    `json:"creatorUid"`
	Timestamp  time.Time `json:"timestamp"`
	Data       string    `json:"data"`
	Details    struct {
		Type        string `json:"type"`
		Callsign    string `json:"callsign"`
		IconsetPath string `json:"iconsetPath"`
		Color       string `json:"color"`
		Location    struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"location"`
	} `json:"details"`
}

func GetDefaultMission(name string) *Mission {
	m := new(Mission)

	m.Name = name
	m.DefaultRole.Type = "MISSION_SUBSCRIBER"
	m.DefaultRole.Permissions = []string{"MISSION_WRITE", "MISSION_READ"}
	m.OwnerRole.Type = "MISSION_OWNER"
	m.OwnerRole.Permissions = []string{"MISSION_MANAGE_FEEDS", "MISSION_SET_PASSWORD", "MISSION_WRITE", "MISSION_MANAGE_LAYERS", "MISSION_UPDATE_GROUPS", "MISSION_READ", "MISSION_DELETE", "MISSION_SET_ROLE"}

	return m
}
