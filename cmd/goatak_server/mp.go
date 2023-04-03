package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	infoFileName = "info.json"
)

type PackageManager struct {
	logger  *zap.SugaredLogger
	baseDir string
	data    sync.Map
}

func NewPackageManager(logger *zap.SugaredLogger) *PackageManager {
	return &PackageManager{
		logger:  logger,
		baseDir: "./data",
		data:    sync.Map{},
	}
}

type PackageInfo struct {
	UID                string    `json:"UID"`
	SubmissionDateTime time.Time `json:"SubmissionDateTime"`
	Keywords           []string  `json:"Keywords"`
	MIMEType           string    `json:"MIMEType"`
	Size               int64     `json:"Size"`
	SubmissionUser     string    `json:"SubmissionUser"`
	PrimaryKey         int       `json:"PrimaryKey"`
	Hash               string    `json:"Hash"`
	CreatorUID         string    `json:"CreatorUid"`
	Name               string    `json:"Name"`
	Tool               string    `json:"Tool"`
}

func (pm *PackageManager) Init() error {
	files, err := os.ReadDir(pm.baseDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		hash := f.Name()
		if pi, err := loadInfo(pm.baseDir, hash); err == nil {
			pm.data.Store(hash, pi)
		} else {
			pm.logger.Errorf("error loading info for hash %s: %s", hash, err.Error())
		}
	}

	return nil
}

func (pm *PackageManager) Put(hash string, pi *PackageInfo) {
	pm.data.Store(hash, pi)

	if err := saveInfo(pm.baseDir, hash, pi); err != nil {
		pm.logger.Errorf("%v", err)
	}
}

func (pm *PackageManager) Get(hash string) (*PackageInfo, bool) {
	i, ok := pm.data.Load(hash)
	if ok {
		return i.(*PackageInfo), ok
	} else {
		return nil, ok
	}
}

func (pm *PackageManager) Range(f func(key string, pi *PackageInfo) bool) {
	pm.data.Range(func(key, value any) bool {
		return f(key.(string), value.(*PackageInfo))
	})
}

func saveInfo(baseDir, hash string, finfo *PackageInfo) error {
	fn, err := os.Create(filepath.Join(baseDir, hash, infoFileName))
	if err != nil {
		return err
	}
	defer fn.Close()

	enc := json.NewEncoder(fn)

	return enc.Encode(finfo)
}

func loadInfo(baseDir, hash string) (*PackageInfo, error) {
	fname := filepath.Join(baseDir, hash, infoFileName)

	if !exists(fname) {
		return nil, fmt.Errorf("info file %s does not exists", fname)
	}

	pi := new(PackageInfo)

	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	err = dec.Decode(pi)

	if err != nil {
		return pi, err
	}
	if pi.Tool == "" {
		pi.Tool = "public"
	}

	return pi, nil
}

func (pm *PackageManager) GetFilePath(hash string) string {
	if pi, ok := pm.Get(hash); ok {
		return filepath.Join(pm.baseDir, hash, pi.Name)
	}
	return ""
}

func (pm *PackageManager) SaveFile(hash, fname string, reader io.Reader) (int64, error) {
	dir := filepath.Join(pm.baseDir, hash)
	if !exists(dir) {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return 0, err
		}
	}

	fn, err := os.Create(filepath.Join(dir, fname))
	if err != nil {
		return 0, err
	}
	defer fn.Close()

	return io.Copy(fn, reader)
}
