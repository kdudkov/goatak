default: all

.PHONY: all
all: dep test build

GIT_REVISION=$(shell git describe --always --dirty)
GIT_COMMIT=$(shell git rev-parse --short HEAD)

LDFLAGS=-ldflags "-s -X main.gitRevision=$(GIT_REVISION) -X main.gitCommit=$(GIT_COMMIT)"

.PHONY: clean
clean:
	rm bin/* || true

.PHONY: dep
dep:
	go mod tidy

.PHONY: protoc
protoc:
	protoc -I=./protobuf --go_out=./cot/v1 --go_opt=module=github.com/kdudkov/goatak/cot/v1 ./protobuf/*.proto

.PHONY: checkdep
checkdep:
	go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all 2> /dev/null

.PHONY: test
test:
	go test ./...

.PHONY: build
build: clean dep
	[ -d dist ] || mkdir dist
	go build $(LDFLAGS) -o dist/ ./cmd/...
