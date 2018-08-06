all: build

build:
	go generate
	go build -tags=deploy_build .

install:
	go generate
	go install -tags=deploy_build .

test: build
	go test ./...

pack-webui:
	npm run --prefix webui build
	go run webui/pack_webui.go

.PHONY: build install test pack-webui
