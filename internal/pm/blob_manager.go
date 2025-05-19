package pm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kdudkov/goatak/pkg/util"
)

var NotFound = fmt.Errorf("blob is not found")

type BlobManager struct {
	logger  *slog.Logger
	mx      sync.RWMutex
	basedir string
}

func NewBlobManages(basedir string) *BlobManager {
	_ = os.MkdirAll(basedir, 0777)

	return &BlobManager{
		logger:  slog.With("logger", "file_manager"),
		mx:      sync.RWMutex{},
		basedir: basedir,
	}
}

func (m *BlobManager) fileName(scope, hash string) string {
	if scope == "" {
		return filepath.Join(m.basedir, hash)
	}

	_ = os.MkdirAll(filepath.Join(m.basedir, scope), 0777)

	return filepath.Join(m.basedir, scope, hash)
}

func (m *BlobManager) GetFile(hash string, scope string) (io.ReadSeekCloser, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	if hash == "" || !util.FileExists(m.fileName(scope, hash)) {
		return nil, NotFound
	}

	return os.Open(m.fileName(scope, hash))
}

func (m *BlobManager) GetFileStat(scope, hash string) (os.FileInfo, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	if hash == "" {
		return nil, fmt.Errorf("no hash")
	}

	return os.Stat(m.fileName(scope, hash))
}

func (m *BlobManager) PutFile(scope, hash string, r io.Reader) (string, int64, error) {
	if r == nil {
		return "", 0, fmt.Errorf("no reader")
	}

	m.mx.Lock()
	defer m.mx.Unlock()

	if hash != "" && util.FileExists(m.fileName(scope, hash)) {
		return hash, 0, nil
	}

	f, err := os.CreateTemp("", "")

	if err != nil {
		return "", 0, err
	}

	h := sha256.New()

	rr := io.TeeReader(r, h)

	var n int64

	if n, err = io.Copy(f, rr); err != nil {
		return "", 0, err
	}

	hash1 := hex.EncodeToString(h.Sum(nil))

	if hash != "" && hash != hash1 {
		return "", 0, fmt.Errorf("invalid hash")
	}

	if err1 := f.Close(); err1 != nil {
		return "", 0, err1
	}

	err = os.Rename(f.Name(), m.fileName(scope, hash1))

	return hash1, n, err
}
