package repository

import (
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kdudkov/goatak/cmd/goatak_server/database"
	"github.com/kdudkov/goatak/internal/cache"
	"github.com/kdudkov/goatak/pkg/model"
)

var _ UserRepository = &UserDbRepository{}

type UserDbRepository struct {
	logger   *slog.Logger
	userFile string
	cache    *cache.Cache[*model.Device]
	dbm      *database.DatabaseManager
}

func NewUserDbRepository(userFile string, dbm *database.DatabaseManager) *UserDbRepository {
	u := &UserDbRepository{
		userFile: userFile,
		logger:   slog.With(slog.String("logger", "user_repo")),
		dbm:      dbm,
	}

	u.cache = cache.NewWithTTL[*model.Device](time.Second*10, u.loadUser)

	return u
}

func (u UserDbRepository) loadUser(username string) *model.Device {
	return u.dbm.DeviceQuery().Login(username).One()
}

func (u UserDbRepository) Start() error {
	if u.dbm.DeviceQuery().Count() == 0 {
		if err := u.loadUsersFile(); err != nil {
			return err
		}
	}

	return nil
}

func (u UserDbRepository) Stop() {
	//no-op
}

func (u UserDbRepository) CheckAuth(username, password string) bool {
	user := u.cache.Load(username)

	if user.Disabled {
		return false
	}

	return user.CheckPassword(password)
}

func (u UserDbRepository) IsValid(username, sn string) bool {
	user := u.cache.Load(username)

	return user != nil && !user.Disabled
}

func (u UserDbRepository) Get(username string) *model.Device {
	return u.cache.Load(username)
}

func (u UserDbRepository) loadUsersFile() error {
	if _, err := os.Lstat(u.userFile); os.IsNotExist(err) {
		return nil
	}

	dat, err := os.ReadFile(u.userFile)
	if err != nil {
		return err
	}

	users := make([]*model.Device, 0)

	if err1 := yaml.Unmarshal(dat, &users); err1 != nil {
		return err1
	}

	for _, user := range users {
		if user.Login != "" {
			if err1 := u.dbm.Save(user); err1 != nil {
				return err1
			}
		}
	}

	return nil
}

func (u UserDbRepository) SaveSignInfo(username string, uid, sn string, till time.Time) {
	cert := &model.Certificate{
		UID:       uid,
		Login:     username,
		Serial:    sn,
		ValidTill: &till,
	}

	u.dbm.Save(cert)
}

func (u UserDbRepository) SaveConnectInfo(username string, sn string) {
	_ = u.dbm.DeviceQuery().Login(username).Update(map[string]any{"last_connect": time.Now()})
	_ = u.dbm.CertsQuery().SN(sn).Update(map[string]any{"last_connect": time.Now()})
}
