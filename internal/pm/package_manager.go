package pm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/google/uuid"
)

const (
	infoFileName = "info.json"
)

type PackageManager struct {
	logger  *slog.Logger
	baseDir string
	data    sync.Map
	noSave  bool
}

func NewPackageManager(basedir string) *PackageManager {
	return &PackageManager{
		logger:  slog.Default().With("logger", "package_manager"),
		baseDir: basedir,
		data:    sync.Map{},
	}
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
			pm.logger.Error("error loading info for "+uid, "error", err.Error())
		}
	}

	return nil
}

func (pm *PackageManager) Stop() {
	// noop
}

func (pm *PackageManager) Store(pi *PackageInfo) {
	pm.data.Store(pi.UID, pi)

	if pm.noSave {
		return
	}

	if err := saveInfo(pm.baseDir, pi); err != nil {
		pm.logger.Error("store error", "error", err.Error())
	}
}

func (pm *PackageManager) Get(uid string) *PackageInfo {
	if i, ok := pm.data.Load(uid); ok {
		return i.(*PackageInfo)
	}

	return nil
}

func (pm *PackageManager) GetByHash(hash string) []*PackageInfo {
	return pm.GetList(func(pi *PackageInfo) bool {
		return pi.Hash == hash
	})
}

func (pm *PackageManager) GetList(filter func(pi *PackageInfo) bool) []*PackageInfo {
	var res []*PackageInfo

	pm.data.Range(func(_, value any) bool {
		if pi, ok := value.(*PackageInfo); ok {
			if filter == nil || filter(pi) {
				res = append(res, pi)
			}
		}

		return true
	})

	return res
}

func (pm *PackageManager) GetFirst(filter func(pi *PackageInfo) bool) *PackageInfo {
	var pi *PackageInfo

	pm.data.Range(func(_, value any) bool {
		p := value.(*PackageInfo)

		if filter == nil || filter(pi) {
			pi = p

			return false
		}

		return true
	})

	return pi
}

func saveInfo(baseDir string, finfo *PackageInfo) error {
	fn, err := os.Create(filepath.Join(baseDir, finfo.UID, infoFileName))
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

func (pm *PackageManager) SaveFile(pi *PackageInfo, data []byte) error {
	hash1 := Hash(data)

	if pi.Hash != "" && pi.Hash != hash1 {
		return fmt.Errorf("bad hash")
	}

	if pi.Name == "" {
		return fmt.Errorf("no name")
	}

	pi.Hash = hash1
	pi.Size = strconv.Itoa(len(data))

	if pi.UID == "" {
		pi.UID = uuid.NewString()
	}

	if !pm.noSave {
		if err := pm.saveFile(pi.UID, pi.Name, data); err != nil {
			return err
		}
	}

	pm.Store(pi)

	return nil
}

func (pm *PackageManager) saveFile(uid, fname string, b []byte) error {
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
