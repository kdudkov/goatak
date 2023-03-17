package main

type ProfileProvider struct {
}

func NewProfileProvider() *ProfileProvider {
	return &ProfileProvider{}
}

func (pp *ProfileProvider) GetProfile(user, uid string) []ZipFile {
	if pp == nil {
		return nil
	}

	return []ZipFile{NewDefaultPrefFile()}
}
