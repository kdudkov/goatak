package pm

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/google/uuid"
)

type PackageManagerFS struct {
	logger  *slog.Logger
	baseDir string
	data    sync.Map
	noSave  bool
	files   *BlobManager
}

func NewPackageManager(basedir string) *PackageManagerFS {
	return &PackageManagerFS{
		logger:  slog.Default().With("logger", "package_manager"),
		baseDir: basedir,
		data:    sync.Map{},
		files:   NewBlobManages(slog.Default().With("logger", "file_manager"), filepath.Join(basedir, "blob")),
	}
}

func (pm *PackageManagerFS) Start() error {
	if err := os.MkdirAll(pm.baseDir, 0777); err != nil {
		return err
	}

	files, err := os.ReadDir(pm.baseDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if !strings.HasSuffix(f.Name(), ".yml") {
			continue
		}

		uid := f.Name()[:len(f.Name())-4]

		if pi, err := loadInfo(pm.baseDir, f.Name()); err == nil {
			pm.data.Store(uid, pi)
		} else {
			pm.logger.Error("error loading info for "+uid, "error", err.Error())
		}
	}

	return nil
}

func (pm *PackageManagerFS) Stop() {
	// noop
}

func (pm *PackageManagerFS) Store(pi *PackageInfo) {
	pm.data.Store(pi.UID, pi)

	if pm.noSave {
		return
	}

	if err := saveInfo(pm.baseDir, pi); err != nil {
		pm.logger.Error("store error", "error", err.Error())
	}
}

func (pm *PackageManagerFS) Get(uid string) *PackageInfo {
	if i, ok := pm.data.Load(uid); ok {
		return i.(*PackageInfo)
	}

	return nil
}

func (pm *PackageManagerFS) GetByHash(hash string) []*PackageInfo {
	return pm.GetList(func(pi *PackageInfo) bool {
		return pi.Hash == hash
	})
}

func (pm *PackageManagerFS) GetList(filter func(pi *PackageInfo) bool) []*PackageInfo {
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

func (pm *PackageManagerFS) GetFirst(filter func(pi *PackageInfo) bool) *PackageInfo {
	var pi *PackageInfo

	pm.data.Range(func(_, value any) bool {
		p := value.(*PackageInfo)

		if filter == nil || filter(p) {
			pi = p

			return false
		}

		return true
	})

	return pi
}

func saveInfo(baseDir string, finfo *PackageInfo) error {
	fn, err := os.Create(filepath.Join(baseDir, finfo.UID+".yml"))
	if err != nil {
		return err
	}
	defer fn.Close()

	enc := yaml.NewEncoder(fn)

	return enc.Encode(finfo)
}

func loadInfo(baseDir, fn string) (*PackageInfo, error) {
	fname := filepath.Join(baseDir, fn)

	if !fileExists(fname) {
		return nil, fmt.Errorf("info file %s does not exists", fname)
	}

	pi := new(PackageInfo)

	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	err = dec.Decode(pi)

	if err != nil {
		return pi, err
	}

	return pi, nil
}

func (pm *PackageManagerFS) GetFile(hash string) (io.ReadSeekCloser, error) {
	return pm.files.GetFile(hash)
}

func (pm *PackageManagerFS) SaveFile(pi *PackageInfo, r io.Reader) error {
	hash1, size, err := pm.files.PutFile(pi.Hash, r)

	if err != nil {
		return err
	}

	pi.Hash = hash1
	if size != 0 {
		pi.Size = int(size)
	}

	if pi.UID == "" {
		pi.UID = uuid.NewString()
	}

	pm.Store(pi)

	return nil
}

func (pm *PackageManagerFS) saveFile(uid, fname string, b []byte) error {
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

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return os.IsExist(err)
	}

	return true
}
