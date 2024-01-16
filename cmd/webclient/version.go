package main

import "fmt"

const unknown = "unknown"

var (
	gitRevision = unknown
	gitBranch   = unknown
)

func getVersion() string {
	if gitBranch != "master" && gitBranch != unknown {
		return fmt.Sprintf("%s:%s", gitBranch, gitRevision)
	}

	return gitRevision
}
