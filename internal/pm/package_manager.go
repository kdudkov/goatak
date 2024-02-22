package pm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	infoFileName = "info.json"
)

type PackageManager struct {
	logger  *zap.SugaredLogger
	baseDir string
	data    sync.Map
	noSave  bool
}

func NewPackageManager(logger *zap.SugaredLogger, basedir string) *PackageManager {
	return &PackageManager{
		logger:  logger,
		baseDir: basedir,
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
	Scope			   string  	 `json:"Scope"`
	Name               string    `json:"Name"`
	Tool               string    `json:"Tool"`
}

func (pm *PackageManager) Start() error {
	if err := os.MkdirAll(pm.baseDir, 0777); err != nil {
		return err
	}

	files, err := os.ReadDir(pm.baseDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}

		uid := f.Name()
		if pi, err := loadInfo(pm.baseDir, uid); err == nil {
			pm.data.Store(uid, pi)
		} else {
			pm.logger.Errorf("error loading info for %s: %s", uid, err.Error())
		}
	}

	return nil
}

func (pm *PackageManager) Stop() {
	// noop
}

func (pm *PackageManager) Store(uid string, pi *PackageInfo) {
	pm.data.Store(uid, pi)

	if pm.noSave {
		return
	}

	if err := saveInfo(pm.baseDir, uid, pi); err != nil {
		pm.logger.Errorf("%v", err)
	}
}

func (pm *PackageManager) Get(uid string) *PackageInfo {
	if i, ok := pm.data.Load(uid); ok {
		return i.(*PackageInfo)
	}

	return nil
}

func (pm *PackageManager) GetByHash(hash string) *PackageInfo {
	var pi *PackageInfo

	pm.data.Range(func(_, value any) bool {
		p := value.(*PackageInfo)

		if p.Hash == hash {
			pi = p

			return false
		}

		return true
	})

	return pi
}

func (pm *PackageManager) ForEach(f func(uid string, pi *PackageInfo) bool) {
	pm.data.Range(func(key, value any) bool {
		return f(key.(string), value.(*PackageInfo))
	})
}

func (pm *PackageManager) GetList(kw, tool string) []*PackageInfo {
	res := make([]*PackageInfo, 0)

	pm.ForEach(func(key string, pi *PackageInfo) bool {
		if tool != "" && tool != pi.Tool {
			return true
		}

		if kw == "" {
			res = append(res, pi)

			return true
		}

		for _, k := range pi.Keywords {
			if kw == k {
				res = append(res, pi)

				return true
			}
		}

		return true
	})

	return res
}

func saveInfo(baseDir, uid string, finfo *PackageInfo) error {
	fn, err := os.Create(filepath.Join(baseDir, uid, infoFileName))
	if err != nil {
		return err
	}
	defer fn.Close()

	enc := json.NewEncoder(fn)

	return enc.Encode(finfo)
}

func loadInfo(baseDir, uid string) (*PackageInfo, error) {
	fname := filepath.Join(baseDir, uid, infoFileName)

	if !fileExists(fname) {
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

	return pi, nil
}

func (pm *PackageManager) GetFilePath(pi *PackageInfo) string {
	return filepath.Join(pm.baseDir, pi.UID, pi.Name)
}

func (pm *PackageManager) SaveData(uid, hash, fname, content string, data []byte, updater func(pi *PackageInfo)) (*PackageInfo, error) {
	hash1 := Hash(data)

	if hash != "" && hash != hash1 {
		return nil, fmt.Errorf("bad hash")
	}

	if uid == "" {
		if pi := pm.GetByHash(hash); pi != nil {
			if pi.Name == fname {
				pm.logger.Infof("file with hash %s exists", hash)
				return pi, nil
			}
		}

		uid = uuid.NewString()
	}

	if !pm.noSave {
		if err := pm.SaveFile(uid, fname, data); err != nil {
			return nil, err
		}
	}

	info := &PackageInfo{
		PrimaryKey:         1,
		UID:                uid,
		SubmissionDateTime: time.Now(),
		Hash:               hash1,
		Name:               fname,
		CreatorUID:         "",
		SubmissionUser:     "",
		Tool:               "",
		Keywords:           nil,
		Size:               int64(len(data)),
		MIMEType:           content,
	}

	if updater != nil {
		updater(info)
	}

	pm.Store(uid, info)
	pm.logger.Infof("save packege %s %s", fname, uid)

	return info, nil
}

func (pm *PackageManager) SaveFile(uid, fname string, b []byte) error {
	dir := filepath.Join(pm.baseDir, uid)
	if !fileExists(dir) {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}
	}

	fn, err := os.Create(filepath.Join(dir, fname))
	if err != nil {
		return err
	}
	defer fn.Close()

	_, err = fn.Write(b)

	return err
}

func Hash(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return os.IsExist(err)
	}

	return true
}
