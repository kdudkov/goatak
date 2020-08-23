default: all

.PHONY: all
all: dep test build

GIT_REVISION=$(shell git describe --always --dirty)
GIT_BRANCH=$(shell git rev-parse --symbolic-full-name --abbrev-ref HEAD)

LDFLAGS=-ldflags "-s -X main.gitRevision=$(GIT_REVISION) -X main.gitBranch=$(GIT_BRANCH)"

.PHONY: clean
clean:
	rm bin/* || true

.PHONY: dep
dep:
	go mod tidy

.PHONY: checkdep
checkdep:
	go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all 2> /dev/null

.PHONY: test
test:
	go test ./...

.PHONY: build
build: clean dep
	[ -d bin ] || mkdir bin
	go build $(LDFLAGS) -o bin/ ./cmd/...

.PHONY: gox
gox: clean dep
	[ -d bin ] || mkdir bin
	GOARM=5 gox --osarch="linux/amd64 linux/arm" -output "bin/{{.Dir}}_{{.OS}}_{{.Arch}}" $(LDFLAGS) ./cmd/...
