MEGACHECK := $(GOPATH)/bin/megacheck
UNAME = $(shell uname -s)
export GOROOT = $(shell go env GOROOT)

test:
	go test ./...

race-test:
	go test -race ./...

$(MEGACHECK):
ifeq ($(UNAME),Darwin)
	curl --silent --location --output $(MEGACHECK) https://github.com/kevinburke/go-tools/releases/download/2018-01-25/megacheck-darwin-amd64
else
	curl --silent --location --output $(MEGACHECK) https://github.com/kevinburke/go-tools/releases/download/2018-01-25/megacheck-linux-amd64
endif
	chmod +x $(MEGACHECK)

lint: $(MEGACHECK)
	$(MEGACHECK) ./...
	go vet ./...
