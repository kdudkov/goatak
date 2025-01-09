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

	"github.com/kdudkov/goatak/pkg/tools"
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

func (m *BlobManager) name(hash string) string {
	return filepath.Join(m.basedir, hash)
}

func (m *BlobManager) GetFile(hash string) (io.ReadSeekCloser, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	if hash == "" || !tools.FileExists(m.name(hash)) {
		return nil, NotFound
	}

	return os.Open(m.name(hash))
}

func (m *BlobManager) GetFileStat(hash string) (os.FileInfo, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	if hash == "" {
		return nil, fmt.Errorf("no hash")
	}

	return os.Stat(m.name(hash))
}

func (m *BlobManager) PutFile(hash string, r io.Reader) (string, int64, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	if hash != "" && tools.FileExists(m.name(hash)) {
		return hash, 0, nil
	}

	f, err := os.CreateTemp("", "")

	if err != nil {
		return "", 0, err
	}

	defer f.Close()

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

	err = os.Rename(f.Name(), m.name(hash1))

	return hash1, n, err
}
