package pm

import "io"

type PackageManager interface {
	Start() error
	Stop()
	Store(pi *PackageInfo)
	Get(uid string) *PackageInfo
	GetList(filter func(pi *PackageInfo) bool) []*PackageInfo
	GetFirst(filter func(pi *PackageInfo) bool) *PackageInfo
	GetFile(hash string) (io.ReadSeekCloser, error)
	GetFileSize(hash string) (int64, error)
	SaveFile(pi *PackageInfo, r io.Reader) error
}
