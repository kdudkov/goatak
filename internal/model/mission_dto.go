package model

import (
	"strings"
	"time"

	"github.com/kdudkov/goatak/internal/pm"
)

type CotTime time.Time

func (x CotTime) MarshalText() ([]byte, error) {
	return []byte(time.Time(x).UTC().Format("2006-01-02T15:04:05.999Z07:00")), nil
}

// UnmarshalText implements the text unmarshaller method.
func (x *CotTime) UnmarshalText(text []byte) error {
	t, err := time.Parse("2006-01-02T15:04:05.999Z07:00", string(text))
	if err != nil {
		return err
	}
	*x = CotTime(t)
	return nil
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
	CreatorUID string  `json:"creatorUid"`
	Timestamp  CotTime `json:"timestamp"`
	Data       DataDTO `json:"data"`
}

type DataDTO struct {
	UID            string   `json:"uid"`
	Keywords       []string `json:"keywords"`
	MimeType       string   `json:"mimeType"`
	Name           string   `json:"name"`
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

type MissionChangeDTO struct {
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

type MissionInvitationDTO struct {
	MissionName string          `json:"mission_name"`
	Invitee     string          `json:"invitee"`
	Type        string          `json:"type"`
	CreatorUID  string          `json:"creator_uid"`
	CreateTime  CotTime         `json:"create_time"`
	Role        *MissionRoleDTO `json:"role"`
}

func ToMissionDTO(m *Mission, pm *pm.PackageManager) *MissionDTO {
	if m == nil {
		return nil
	}

	uids := make([]*MissionItemDTO, len(m.Items)+1)

	for i, item := range m.Items {
		uids[i] = NewItemDTO(item)
	}

	mDTO := &MissionDTO{
		Name:           m.Name,
		CreatorUID:     m.CreatorUID,
		CreateTime:     CotTime(m.CreateTime),
		LastEdit:       CotTime(m.LastEdit),
		BaseLayer:      m.BaseLayer,
		Bbox:           m.Bbox,
		ChatRoom:       m.ChatRoom,
		Classification: m.Classification,
		Contents:       nil,
		DefaultRole:    GetRole("MISSION_SUBSCRIBER"),
		//OwnerRole:         GetRole("MISSION_OWNER"),
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
		Uids:              uids,
	}

	if pm != nil {
		mDTO.Contents = make([]*ContentItemDTO, 0)

		for _, h := range m.GetHashes() {
			if pi := pm.GetByHash(h); pi != nil {
				mDTO.Contents = append(mDTO.Contents, toContentItemDTO(pi))
			}
		}
	}

	return mDTO
}

func toContentItemDTO(pi *pm.PackageInfo) *ContentItemDTO {
	return &ContentItemDTO{
		CreatorUID: pi.CreatorUID,
		Timestamp:  CotTime(pi.SubmissionDateTime),
		Data: DataDTO{
			UID:            pi.UID,
			Name:           pi.Name,
			Keywords:       pi.Keywords,
			MimeType:       pi.MIMEType,
			SubmissionTime: CotTime(pi.SubmissionDateTime),
			Submitter:      pi.SubmissionUser,
			CreatorUID:     pi.CreatorUID,
			Hash:           pi.Hash,
			Size:           int(pi.Size),
		},
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
		Role:       GetRole(s.Role),
	}
}

func ToMissionSubscriptionsDTO(subscriptions []*Subscription) []*MissionSubscriptionDTO {
	res := make([]*MissionSubscriptionDTO, len(subscriptions))

	for i, s := range subscriptions {
		res[i] = ToMissionSubscriptionDTO(s)
	}

	return res
}

func ToMissionInvitationDTO(m *Invitation, name string) *MissionInvitationDTO {
	return &MissionInvitationDTO{
		MissionName: name,
		Invitee:     m.Invitee,
		Type:        m.Typ,
		CreatorUID:  m.CreatorUID,
		CreateTime:  CotTime(m.CreateTime),
		Role:        GetRole(m.Role),
	}
}

func NewChangeDTO(c *Change, name string) *MissionChangeDTO {
	cd := &MissionChangeDTO{
		Type:        c.Type,
		MissionName: name,
		Timestamp:   CotTime(c.CreateTime),
		ServerTime:  CotTime(c.CreateTime),
		CreatorUID:  c.CreatorUID,
		ContentUID:  c.ContentUID,
	}

	if c.ContentUID != "" {
		cd.Details = &MissionDetailsDTO{
			Type:        c.CotType,
			Callsign:    c.Callsign,
			IconsetPath: c.IconsetPath,
			Color:       c.Color,
			Location: &LocationDTO{
				Lat: c.Lat,
				Lon: c.Lon,
			},
		}
	}

	return cd
}

func NewItemDTO(i *DataItem) *MissionItemDTO {
	return &MissionItemDTO{
		CreatorUID: i.CreatorUID,
		Timestamp:  CotTime(i.Timestamp),
		Data:       i.UID,
		Details: &MissionDetailsDTO{
			Type:        i.Type,
			Callsign:    i.Callsign,
			Title:       i.Title,
			IconsetPath: i.IconsetPath,
			Color:       i.Color,
			Location: &LocationDTO{
				Lat: i.Lat,
				Lon: i.Lon,
			},
		},
	}
}

func NewRole(typ string, perms ...string) *MissionRoleDTO {
	return &MissionRoleDTO{
		Type:        typ,
		Permissions: perms,
	}
}

func GetRole(name string) *MissionRoleDTO {
	switch name {
	case "MISSION_OWNER":
		return NewRole(name, "MISSION_MANAGE_FEEDS", "MISSION_SET_PASSWORD",
			"MISSION_WRITE", "MISSION_MANAGE_LAYERS", "MISSION_UPDATE_GROUPS", "MISSION_READ", "MISSION_DELETE",
			"MISSION_SET_ROLE")
	case "MISSION_SUBSCRIBER", "":
		return NewRole("MISSION_SUBSCRIBER", "MISSION_WRITE", "MISSION_READ")
	default:
		return NewRole(name)
	}
}
