package repository

import (
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"

	"github.com/kdudkov/goatak/internal/model"
)

type UserFileRepository struct {
	userFile string
	logger   *zap.SugaredLogger
	users    map[string]*model.User

	watcher *fsnotify.Watcher

	mx sync.RWMutex
}

func NewFileUserRepo(logger *zap.SugaredLogger, userFile string) *UserFileRepository {
	um := &UserFileRepository{
		logger:   logger.Named("UserManager"),
		userFile: userFile,
		users:    make(map[string]*model.User),
		mx:       sync.RWMutex{},
	}

	um.loadUsersFile()

	if len(um.users) == 0 {
		um.logger.Infof("no valid users found -  create one")
		bytes, _ := bcrypt.GenerateFromPassword([]byte("11111"), 14)

		um.users["user"] = &model.User{
			Login:    "user",
			Password: string(bytes),
		}
	}

	return um
}

func (r *UserFileRepository) loadUsersFile() error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if _, err := os.Lstat(r.userFile); os.IsNotExist(err) {
		// create empty file
		f, err := os.Create(r.userFile)
		if err != nil {
			return err
		}
		f.Close()
		return nil
	}

	dat, err := os.ReadFile(r.userFile)

	if err != nil {
		return err
	}

	users := make([]*model.User, 0)

	if err := yaml.Unmarshal(dat, &users); err != nil {
		return err
	}

	r.users = make(map[string]*model.User)
	for _, user := range users {
		if user.Login != "" {
			r.users[user.Login] = user
		}
	}

	return nil
}

func (r *UserFileRepository) Start() error {
	var err error
	r.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := r.watcher.Add(r.userFile); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-r.watcher.Events:
				if !ok {
					return
				}
				r.logger.Debugf("event: %v", event)
				if event.Has(fsnotify.Write) && event.Name == r.userFile {
					r.logger.Infof("users file is modified, reloading")
					if err := r.loadUsersFile(); err != nil {
						r.logger.Errorf("error: %s", err.Error())
					}
				}
			case err, ok := <-r.watcher.Errors:
				if !ok {
					return
				}
				r.logger.Errorf("error: %s", err.Error())
			}
		}
	}()

	return nil
}

func (r *UserFileRepository) Stop() {
	if r.watcher != nil {
		_ = r.watcher.Close()
	}
}

func (r *UserFileRepository) CheckUserAuth(user, password string) bool {
	r.mx.RLock()
	defer r.mx.RUnlock()
	if user, ok := r.users[user]; ok {
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		return err == nil
	}
	return false
}

func (r *UserFileRepository) UserIsValid(user, sn string) bool {
	r.mx.RLock()
	defer r.mx.RUnlock()
	_, ok := r.users[user]
	return ok
}

func (r *UserFileRepository) GetUser(username string) *model.User {
	r.mx.RLock()
	defer r.mx.RUnlock()
	return r.users[username]
}
