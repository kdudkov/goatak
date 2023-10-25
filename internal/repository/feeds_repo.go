package repository

import (
	"github.com/kdudkov/goatak/pkg/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

type FeedsFileRepository struct {
	logger  *zap.SugaredLogger
	baseDir string
}

func NewFeedsFileRepo(logger *zap.SugaredLogger, basedir string) *FeedsFileRepository {
	return &FeedsFileRepository{
		logger:  logger,
		baseDir: basedir,
	}
}

func (r *FeedsFileRepository) Start() error {
	if err := os.MkdirAll(r.baseDir, 0777); err != nil {
		return err
	}

	return nil
}

func (r *FeedsFileRepository) Stop() {
	// noop
}

func (r *FeedsFileRepository) Store(f *model.Feed2) {
	if f == nil || f.Uid == "" {
		return
	}

	fl, err := os.Create(filepath.Join(r.baseDir, f.Uid+".yml"))
	if err != nil {
		r.logger.Errorf("error: %s", err.Error())
		return
	}
	defer fl.Close()

	enc := yaml.NewEncoder(fl)

	if err := enc.Encode(f); err != nil {
		r.logger.Errorf("error: %s", err.Error())
	}
}

func (r *FeedsFileRepository) Get(uid string) *model.Feed2 {
	return r.load(uid + ".yml")
}

func (r *FeedsFileRepository) load(fname string) *model.Feed2 {
	fl, err := os.Open(filepath.Join(r.baseDir, fname))
	if err != nil {
		return nil
	}

	defer fl.Close()

	var f *model.Feed2
	dec := yaml.NewDecoder(fl)

	if err := dec.Decode(&f); err != nil {
		r.logger.Errorf("error: %s", err.Error())
	}

	f.Active = true
	return f
}

func (r *FeedsFileRepository) Remove(uid string) {
	os.Remove(filepath.Join(r.baseDir, uid+".yml"))
}

func (r *FeedsFileRepository) ForEach(f func(item *model.Feed2) bool) {
	files, err := os.ReadDir(r.baseDir)
	if err != nil {
		r.logger.Errorf("error: %s", err.Error())
		return
	}

	for _, fl := range files {
		if fl.IsDir() {
			continue
		}
		if !strings.HasSuffix(fl.Name(), ".yml") {
			continue
		}
		feed := r.load(fl.Name())
		if feed != nil {
			if !f(feed) {
				return
			}
		}
	}
}
