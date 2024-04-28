package pm

import "time"

type PackageInfo struct {
	UID                string    `json:"UID" yaml:"UID"`
	SubmissionDateTime time.Time `json:"SubmissionDateTime" yaml:"time"`
	Keywords           []string  `json:"Keywords" yaml:"keywords"`
	MIMEType           string    `json:"MIMEType" yaml:"MIMEType"`
	Size               int       `json:"Size" yaml:"size"`
	SubmissionUser     string    `json:"SubmissionUser" yaml:"user"`
	PrimaryKey         int       `json:"PrimaryKey" yaml:"-"`
	Hash               string    `json:"Hash" yaml:"hash"`
	CreatorUID         string    `json:"CreatorUid" yaml:"creator_uid"`
	Scope              string    `json:"Scope" yaml:"scope"`
	Name               string    `json:"Name" yaml:"name"`
	Tool               string    `json:"Tool" yaml:"tool"`
}

func (pi *PackageInfo) HasKeyword(kw string) bool {
	if pi == nil {
		return false
	}

	if kw == "" {
		return true
	}

	for _, k := range pi.Keywords {
		if k == kw {
			return true
		}
	}

	return false
}
