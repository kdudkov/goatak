package main

import (
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type UserInfo struct {
	User     string `yaml:"user"`
	Callsign string `yaml:"callsign"`
	Team     string `yaml:"team"`
	Role     string `yaml:"role"`
	Typ      string `yaml:"type"`
	Password string `yaml:"password"`
	Scope    string `yaml:"scope"`
}

type UserManager struct {
	userFile string
	logger   *zap.SugaredLogger
	users    map[string]*UserInfo

	watcher *fsnotify.Watcher

	mx sync.RWMutex
}

func NewUserManager(logger *zap.SugaredLogger, userFile string) *UserManager {
	um := &UserManager{
		logger:   logger.Named("UserManager"),
		userFile: userFile,
		users:    make(map[string]*UserInfo),
		mx:       sync.RWMutex{},
	}

	um.loadUsersFile()

	if len(um.users) == 0 {
		um.logger.Infof("no valid users found -  create one")
		bytes, _ := bcrypt.GenerateFromPassword([]byte("11111"), 14)

		um.users["user"] = &UserInfo{
			User:     "user",
			Password: string(bytes),
		}
	}

	return um
}

func (um *UserManager) loadUsersFile() error {
	um.mx.Lock()
	defer um.mx.Unlock()

	dat, err := os.ReadFile(um.userFile)

	if err != nil {
		return err
	}

	users := make([]*UserInfo, 0)

	if err := yaml.Unmarshal(dat, &users); err != nil {
		return err
	}

	um.users = make(map[string]*UserInfo)
	for _, user := range users {
		if user.User != "" {
			um.users[user.User] = user
		}
	}

	return nil
}

func (um *UserManager) Start() error {
	var err error
	um.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	um.watcher.Add(um.userFile)
	go func() {
		for {
			select {
			case event, ok := <-um.watcher.Events:
				if !ok {
					return
				}
				um.logger.Debugf("event: %v", event)
				if event.Has(fsnotify.Write) && event.Name == um.userFile {
					um.logger.Infof("users file is modified, reloading")
					if err := um.loadUsersFile(); err != nil {
						um.logger.Errorf("error: %s", err.Error())
					}
				}
			case err, ok := <-um.watcher.Errors:
				if !ok {
					return
				}
				um.logger.Errorf("error: %s", err.Error())
			}
		}
	}()

	return nil
}

func (um *UserManager) Stop() {
	if um.watcher != nil {
		_ = um.watcher.Close()
	}
}

func (um *UserManager) CheckUserAuth(user, password string) bool {
	um.mx.RLock()
	defer um.mx.RUnlock()
	if user, ok := um.users[user]; ok {
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		return err == nil
	}
	return false
}

func (um *UserManager) UserIsValid(user, sn string) bool {
	um.mx.RLock()
	defer um.mx.RUnlock()
	_, ok := um.users[user]
	return ok
}

func (um *UserManager) GetUser(username string) *UserInfo {
	um.mx.RLock()
	defer um.mx.RUnlock()
	return um.users[username]
}

func (um *UserManager) GetProfile(user, uid string) []ZipFile {
	um.mx.RLock()
	defer um.mx.RUnlock()
	if um == nil {
		return nil
	}
	res := make([]ZipFile, 0)

	if user != "" {
		if user, ok := um.users[user]; ok {
			if user.Callsign != "" || user.Team != "" || user.Role != "" || user.Typ != "" {
				res = append(res, NewUserPrefsFile(user.Callsign, user.Team, user.Role, user.Typ))
			}
		}
	}

	if f, err := NewFsFile("defaults.pref"); err == nil {
		res = append(res, f)
	}

	if paths, err := os.ReadDir(filepath.Join(baseDir, "maps")); err == nil {
		for _, p := range paths {
			if !p.IsDir() && strings.HasSuffix(p.Name(), ".xml") {
				if f, err := NewFsFile(filepath.Join("maps", p.Name())); err == nil {
					res = append(res, f)
				}
			}
		}
	}

	return res
}

func NewUserPrefsFile(callsign, team, role, typ string) *PrefFile {
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
	if typ != "" {
		conf.AddParam("locationUnitType", typ)
	}
	return conf
}
