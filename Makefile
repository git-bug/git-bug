all: build

UNAME_S := $(shell uname -s)
XARGS:=xargs -r
ifeq ($(UNAME_S),Darwin)
    XARGS:=xargs
endif

COMMANDS_PATH:=github.com/MichaelMure/git-bug/commands

.PHONY: build
build:
	go generate
	go build .

# produce a build debugger friendly
.PHONY: debug-build
debug-build:
	go generate
	go build -gcflags=all="-N -l" .

.PHONY: install
install:
	go generate
	go install .

.PHONY: releases
releases:
	go generate
	go run github.com/mitchellh/gox@v1.0.1 -osarch '!darwin/386' -output "dist/{{.Dir}}_{{.OS}}_{{.Arch}}"

secure: secure-practices secure-vulnerabilities

.PHONY: secure-practices
secure-practices:
# TODO: change pinned version of GoKart to "latest" once PR #84 is merged
#       https://github.com/praetorian-inc/gokart/pull/84
# go install github.com/praetorian-inc/gokart@latest
	go install github.com/selesy/gokart-pre
	gokart scan

.PHONY: secure-vulnerabilities
secure-vulnerabilities:
	go install golang.org/x/vuln/cmd/govulncheck
	govulncheck ./... 

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
	go build -tags=debugwebui

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
