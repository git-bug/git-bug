all: build

GIT_COMMIT:=$(shell git rev-list -1 HEAD)
GIT_LAST_TAG:=$(shell git describe --abbrev=0 --tags)
GIT_EXACT_TAG:=$(shell git name-rev --name-only --tags HEAD)
UNAME_S := $(shell uname -s)
XARGS:=xargs -r
ifeq ($(UNAME_S),Darwin)
    XARGS:=xargs
endif

COMMANDS_PATH:=github.com/git-bug/git-bug/commands
LDFLAGS:=-X ${COMMANDS_PATH}.GitCommit=${GIT_COMMIT} \
	-X ${COMMANDS_PATH}.GitLastTag=${GIT_LAST_TAG} \
	-X ${COMMANDS_PATH}.GitExactTag=${GIT_EXACT_TAG}

.PHONY: build
build:
	go generate
	go build -ldflags "$(LDFLAGS)" .

# produce a build debugger friendly
.PHONY: debug-build
debug-build:
	go generate
	go build -ldflags "$(LDFLAGS)" -gcflags=all="-N -l" .

.PHONY: install
install:
	go generate
	go install -ldflags "$(LDFLAGS)" .

.PHONY: releases
releases:
	go generate
	go run github.com/mitchellh/gox@v1.0.1 -ldflags "$(LDFLAGS)" -osarch '!darwin/386' -output "dist/{{.Dir}}_{{.OS}}_{{.Arch}}"

secure: secure-practices secure-vulnerabilities

.PHONY: secure-practices
secure-practices:
	go run github.com/praetorian-inc/gokart scan

.PHONY: secure-vulnerabilities
secure-vulnerabilities:
	go run golang.org/x/vuln/cmd/govulncheck ./...

.PHONY: test
test:
	go test -v -bench=. ./...

.PHONY: pack-webui
pack-webui:
	npm run --prefix webui build
	go run webui/pack_webui.go

# produce a build that will fetch the web UI from the filesystem instead of from the binary
.PHONY: debug-webui
debug-webui:
	go build -ldflags "$(LDFLAGS)" -tags=debugwebui

.PHONY: clean-local-bugs
clean-local-bugs:
	git for-each-ref refs/bugs/ | cut -f 2 | $(XARGS) -n 1 git update-ref -d
	git for-each-ref refs/remotes/origin/bugs/ | cut -f 2 | $(XARGS) -n 1 git update-ref -d
	rm -f .git/git-bug/bug-cache

.PHONY: clean-remote-bugs
clean-remote-bugs:
	git ls-remote origin "refs/bugs/*" | cut -f 2 | $(XARGS) git push origin -d

.PHONY: clean-local-identities
clean-local-identities:
	git for-each-ref refs/identities/ | cut -f 2 | $(XARGS) -n 1 git update-ref -d
	git for-each-ref refs/remotes/origin/identities/ | cut -f 2 | $(XARGS) -n 1 git update-ref -d
	rm -f .git/git-bug/identity-cache

.PHONY: clean-local-identities
clean-remote-identities:
	git ls-remote origin "refs/identities/*" | cut -f 2 | $(XARGS) git push origin -d
