all: build

build:
	go generate
	go build -tags=deploy_build .

install:
	go generate
	go install -tags=deploy_build .

test: build
	go test ./...

.PHONY: build install test
