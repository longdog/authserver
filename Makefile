NAME=auth
VERSION=`cat $(VERSION)`
BUILD=$(shell git rev-list -1 HEAD)

ifeq ($(GOOS),windows)
  ext=.exe
else
  ext=
endif


.PHONY: default
default: build

run:
	go run -race -tags debug ./app/main.go -port 3003

build:
	go get -t ./...
	go mod vendor
	go generate ./app
	GO111MODULE=on CGO_ENABLED=0 go build -ldflags "-X main.version=$(VERSION)-$(BUILD)" -mod=vendor -o $(NAME)$(ext) ./app/main.go

test:
	go test -v -count 1 -race -cover -tags mock ./...