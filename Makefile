default: all

.PHONY: all
all: dep test build

GIT_REVISION=$(shell git describe --always --dirty)
GIT_BRANCH=$(shell git rev-parse --symbolic-full-name --abbrev-ref HEAD)

LDFLAGS=-ldflags "-s -X main.gitRevision=$(GIT_REVISION) -X main.gitBranch=$(GIT_BRANCH)"

.PHONY: clean
clean:
	[ -d dist ] || mkdir dist
	rm -rf dist/* || true

.PHONY: dep
dep:
	go mod tidy

.PHONE: update
update:
	rm go.sum; go get -u ./...

.PHONY: protoc
protoc:
	protoc -I=./protobuf --go_out=./cotproto --go_opt=module=github.com/kdudkov/goatak/cotproto ./protobuf/*.proto

.PHONY: checkdep
checkdep:
	go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all 2> /dev/null

.PHONY: test
test:
	go test ./...

.PHONY: build
build: clean dep
	go build $(LDFLAGS) -o dist/ ./cmd/...

.PHONY: gox
gox: clean dep
	GOARM=5 gox --osarch="linux/amd64 windows/amd64 darwin/arm64" -output "dist/{{.OS}}_{{.Arch}}/{{.Dir}}" $(LDFLAGS) ./cmd/...