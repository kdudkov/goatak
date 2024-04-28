package pm

import "io"

type PackageManager interface {
	Start() error
	Stop()
	Store(pi *PackageInfo)
	Get(uid string) *PackageInfo
	GetByHash(hash string) []*PackageInfo
	GetList(filter func(pi *PackageInfo) bool) []*PackageInfo
	GetFirst(filter func(pi *PackageInfo) bool) *PackageInfo
	GetFile(pi *PackageInfo) (io.ReadSeekCloser, error)
	SaveFile(pi *PackageInfo, r io.Reader) error
}
