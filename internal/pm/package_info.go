package pm

import "time"

type PackageInfo struct {
	UID                string    `json:"UID"`
	SubmissionDateTime time.Time `json:"SubmissionDateTime"`
	Keywords           []string  `json:"Keywords"`
	MIMEType           string    `json:"MIMEType"`
	Size               int       `json:"Size"`
	SubmissionUser     string    `json:"SubmissionUser"`
	PrimaryKey         int       `json:"PrimaryKey"`
	Hash               string    `json:"Hash"`
	CreatorUID         string    `json:"CreatorUid"`
	Scope              string    `json:"Scope"`
	Name               string    `json:"Name"`
	Tool               string    `json:"Tool"`
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
