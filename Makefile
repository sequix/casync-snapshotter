GIT_COMMIT := $(shell git rev-parse HEAD)

all: casnap-conv

casnap-conv:
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X 'github.com/sequix/casync-snapshotter/pkg/buildinfo.Commit=$(GIT_COMMIT)'" -o out/casnap-conv cmd/convert/main.go

.PHONY: clean
clean:
	rm -rf out logs tmp