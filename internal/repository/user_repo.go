package repository

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"

	"github.com/kdudkov/goatak/internal/model"
)

type UserFileRepository struct {
	userFile string
	logger   *slog.Logger
	users    map[string]*model.User

	watcher *fsnotify.Watcher

	mx sync.RWMutex
}

func NewFileUserRepo(userFile string) *UserFileRepository {
	um := &UserFileRepository{
		logger:   slog.Default().With("logger", "UserManager"),
		userFile: userFile,
		users:    make(map[string]*model.User),
		mx:       sync.RWMutex{},
	}

	if err := um.loadUsersFile(); err != nil {
		um.logger.Error("error loading users file", "error", err.Error())
	}

	if len(um.users) == 0 {
		um.logger.Info("no valid users found -  create one")

		const bcryptCost = 14
		bytes, _ := bcrypt.GenerateFromPassword([]byte("11111"), bcryptCost)

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

		return f.Close()
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

				r.logger.Debug(fmt.Sprintf("event: %v", event))

				if event.Has(fsnotify.Write) && event.Name == r.userFile {
					r.logger.Info("users file is modified, reloading")

					if err := r.loadUsersFile(); err != nil {
						r.logger.Error("error", "error", err.Error())
					}
				}
			case err, ok := <-r.watcher.Errors:
				if !ok {
					return
				}

				r.logger.Error("error", "error", err.Error())
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
		if err != nil {
			r.logger.Debug("password check failed", "error", err)
			return false
		}
		return true
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
