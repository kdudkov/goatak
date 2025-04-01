package model

import (
	"strconv"
	"strings"
	"time"
)

const cotFormat = "2006-01-02T15:04:05.999Z07:00"

type CotTime time.Time

func (x CotTime) MarshalText() ([]byte, error) {
	return []byte(time.Time(x).UTC().Format(cotFormat)), nil
}

// UnmarshalText implements the text unmarshaller method.
func (x *CotTime) UnmarshalText(text []byte) error {
	t, err := time.Parse(cotFormat, string(text))
	if err != nil {
		return err
	}
	*x = CotTime(t)
	return nil
}

type MissionDTO struct {
	Name              string             `json:"name"`
	Scope             string             `json:"scope,omitempty"`
	CreatorUID        string             `json:"creatorUid"`
	CreateTime        CotTime            `json:"createTime"`
	LastEdit          CotTime            `json:"lastEdited"`
	BaseLayer         string             `json:"baseLayer"`
	Bbox              string             `json:"bbox"`
	ChatRoom          string             `json:"chatRoom"`
	Classification    string             `json:"classification"`
	DefaultRole       *MissionRoleDTO    `json:"defaultRole,omitempty"`
	OwnerRole         *MissionRoleDTO    `json:"ownerRole,omitempty"`
	Description       string             `json:"description"`
	Expiration        int                `json:"expiration"`
	ExternalData      []any              `json:"externalData"`
	Feeds             []string           `json:"feeds"`
	Groups            []string           `json:"groups,omitempty"`
	InviteOnly        bool               `json:"inviteOnly"`
	Keywords          []string           `json:"keywords"`
	MapLayers         []string           `json:"mapLayers"`
	PasswordProtected bool               `json:"passwordProtected"`
	Path              string             `json:"path"`
	Tool              string             `json:"tool"`
	Uids              []*MissionPointDTO `json:"uids"`
	Contents          []*ContentItemDTO  `json:"contents"`
	Token             string             `json:"token"`
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

type MissionPointDTO struct {
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
	Token      string          `json:"token,omitempty"`
}

type MissionChangeDTO struct {
	Type            string             `json:"type"`
	MissionName     string             `json:"missionName"`
	Timestamp       CotTime            `json:"timestamp"`
	CreatorUID      string             `json:"creatorUid"`
	ServerTime      CotTime            `json:"serverTime"`
	ContentUID      string             `json:"contentUid,omitempty"`
	ContentHash     string             `json:"contentHash,omitempty"`
	Details         *MissionDetailsDTO `json:"details,omitempty"`
	ContentResource *ResourceDTO       `json:"contentResource,omitempty"`
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

func ToMissionDTO(m *Mission, withToken bool) *MissionDTO {
	return ToMissionDTOFull(m, withToken, false)
}

func ToMissionDTOAdm(m *Mission) *MissionDTO {
	return ToMissionDTOFull(m, false, true)
}

func ToMissionDTOFull(m *Mission, withToken bool, withScope bool) *MissionDTO {
	if m == nil {
		return nil
	}

	mDTO := &MissionDTO{
		Name:              m.Name,
		CreatorUID:        m.CreatorUID,
		CreateTime:        CotTime(m.CreatedAt),
		LastEdit:          CotTime(m.UpdatedAt),
		BaseLayer:         m.BaseLayer,
		Bbox:              m.Bbox,
		ChatRoom:          m.ChatRoom,
		Classification:    m.Classification,
		DefaultRole:       GetRole("MISSION_SUBSCRIBER"),
		OwnerRole:         GetRole("MISSION_OWNER"),
		Description:       m.Description,
		Expiration:        -1,
		ExternalData:      []any{},
		Feeds:             []string{},
		InviteOnly:        m.InviteOnly,
		Keywords:          strings.Split(m.Keywords, ","),
		MapLayers:         []string{},
		PasswordProtected: m.Password != "",
		Path:              m.Path,
		Tool:              m.Tool,
		Uids:              make([]*MissionPointDTO, len(m.Points)),
		Contents:          make([]*ContentItemDTO, len(m.Resources)),
	}

	for i, p := range m.Points {
		mDTO.Uids[i] = ToMissionPointDTO(p)
	}

	for i, item := range m.Resources {
		mDTO.Contents[i] = ToContentItemDTO(item)
	}

	if withToken {
		mDTO.Token = m.Token
	}

	if withScope {
		mDTO.Scope = m.Scope
	}

	return mDTO
}

func ToMissionSubscriptionDTO(s *Subscription, token string) *MissionSubscriptionDTO {
	if s == nil {
		return nil
	}

	return &MissionSubscriptionDTO{
		ClientUID:  s.ClientUID,
		Username:   s.Username,
		CreateTime: CotTime(s.CreatedAt),
		Role:       GetRole(s.Role),
		Token:      token,
	}
}

func ToMissionSubscriptionsDTO(subscriptions []*Subscription) []*MissionSubscriptionDTO {
	res := make([]*MissionSubscriptionDTO, len(subscriptions))

	for i, s := range subscriptions {
		res[i] = ToMissionSubscriptionDTO(s, "")
	}

	return res
}

func ToMissionInvitationDTO(m *Invitation, name string) *MissionInvitationDTO {
	return &MissionInvitationDTO{
		MissionName: name,
		Invitee:     m.Invitee,
		Type:        m.Typ,
		CreatorUID:  m.CreatorUID,
		CreateTime:  CotTime(m.CreatedAt),
		Role:        GetRole(m.Role),
	}
}

type ResourceDTO struct {
	ID                 string    `json:"PrimaryKey"`
	UID                string    `json:"UID"`
	SubmissionDateTime time.Time `json:"SubmissionDateTime"`
	Keywords           []string  `json:"Keywords"`
	MIMEType           string    `json:"MIMEType"`
	Size               int       `json:"Size"`
	SubmissionUser     string    `json:"SubmissionUser"`
	Hash               string    `json:"Hash"`
	CreatorUID         string    `json:"CreatorUid"`
	FileName           string    `json:"FileName"`
	Name               string    `json:"Name"`
	Tool               string    `json:"Tool"`
	Expiration         int64     `json:"Expiration"`
}

func ToChangeDTO(c *Change, name string) *MissionChangeDTO {
	cd := &MissionChangeDTO{
		Type:        c.Type,
		MissionName: name,
		Timestamp:   CotTime(c.CreatedAt),
		ServerTime:  CotTime(c.CreatedAt),
		CreatorUID:  c.CreatorUID,
		ContentUID:  c.ContentUID,
		ContentHash: c.ContentHash,
	}

	if p := c.MissionPoint; p != nil {
		cd.Details = &MissionDetailsDTO{
			Type:        p.Type,
			Callsign:    p.Callsign,
			IconsetPath: p.IconsetPath,
			Color:       p.Color,
			Location: &LocationDTO{
				Lat: p.Lat,
				Lon: p.Lon,
			},
		}
	}

	if r := c.Resource; r != nil {
		cd.ContentResource = ToResourceDTO(r)
	}

	return cd
}

func ToMissionPointDTO(i *Point) *MissionPointDTO {
	return &MissionPointDTO{
		CreatorUID: i.CreatorUID,
		Timestamp:  CotTime(i.CreatedAt),
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

func ToContentItemDTO(r *Resource) *ContentItemDTO {
	return &ContentItemDTO{
		CreatorUID: r.CreatorUID,
		Timestamp:  CotTime(r.CreatedAt),
		Data: DataDTO{
			UID:            r.UID,
			Keywords:       r.Kw.List(),
			MimeType:       r.MIMEType,
			Name:           r.FileName,
			SubmissionTime: CotTime(r.CreatedAt),
			Submitter:      r.SubmissionUser,
			CreatorUID:     r.CreatorUID,
			Hash:           r.Hash,
			Size:           r.Size,
		},
	}
}

func ToResourceDTO(r *Resource) *ResourceDTO {
	if r == nil {
		return nil
	}

	return &ResourceDTO{
		ID:                 strconv.Itoa(int(r.ID)),
		UID:                r.UID,
		SubmissionDateTime: r.CreatedAt,
		Keywords:           r.Kw.List(),
		MIMEType:           r.MIMEType,
		Size:               r.Size,
		SubmissionUser:     r.SubmissionUser,
		Hash:               r.Hash,
		CreatorUID:         r.CreatorUID,
		FileName:           r.FileName,
		Name:               r.Name,
		Tool:               r.Tool,
		Expiration:         r.Expiration,
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
