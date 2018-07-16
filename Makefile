all: build

build:
	go generate
	go build -tags=deploy_build .

install:
	go generate
	go install -tags=deploy_build .

.PHONY: build install
