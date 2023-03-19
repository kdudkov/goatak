package main

import (
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
	"os"
)

type UserInfo struct {
	User     string `yaml:"user"`
	Callsign string `yaml:"callsign"`
	Team     string `yaml:"team"`
	Role     string `yaml:"role"`
	Password string `yaml:"password"`
}

type UserManager struct {
	userFile string
	logger   *zap.SugaredLogger
	users    map[string]*UserInfo
}

func NewUserManager(logger *zap.SugaredLogger, userFile string) *UserManager {
	dat, err := os.ReadFile("users.yml")

	usermap := make(map[string]*UserInfo)
	if err == nil {
		users := make([]*UserInfo, 0)
		yaml.Unmarshal(dat, &users)

		for _, user := range users {
			if user.User != "" {
				usermap[user.User] = user
			}
		}
	}

	if len(usermap) == 0 {
		bytes, _ := bcrypt.GenerateFromPassword([]byte("11111"), 14)

		usermap["user"] = &UserInfo{
			User:     "user",
			Password: string(bytes),
		}
	}
	return &UserManager{
		logger:   logger,
		userFile: userFile,
		users:    usermap,
	}
}

func (um *UserManager) CheckUserAuth(user, password string) bool {
	if user, ok := um.users[user]; ok {
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		return err == nil
	}
	return false
}

func (um *UserManager) UserIsValid(user, sn string) bool {
	_, ok := um.users[user]
	return ok
}

func (um *UserManager) GetProfile(user, uid string) []ZipFile {
	if um == nil {
		return nil
	}
	res := make([]ZipFile, 0)

	if user, ok := um.users[user]; ok {
		if user.Callsign != "" || user.Team != "" || user.Role != "" {
			res = append(res, NewUserFrefsFile(user.Callsign, user.Team, user.Role))
		}
	}

	if f, err := NewFsFile("defaults.pref"); err == nil {
		res = append(res, f)
	}
	return res
}

func NewUserFrefsFile(callsign, team, role string) *PrefFile {
	conf := NewUserProfilePrefFile()
	if callsign != "" {
		conf.AddParam("locationCallsign", callsign)
	}
	if team != "" {
		conf.AddParam("locationTeam", team)
	}
	if role != "" {
		conf.AddParam("atakRoleType", role)
	}
	return conf
}

func NewUserFrefsFileWithType(callsign, team, role, typ string) *PrefFile {
	conf := NewUserProfilePrefFile()
	conf.AddParam("locationCallsign", callsign)
	conf.AddParam("locationTeam", team)
	conf.AddParam("atakRoleType", role)
	conf.AddParam("locationUnitType", typ)
	return conf
}
