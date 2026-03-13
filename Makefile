XARGS:=xargs -r
ifeq ($(shell uname -s),Darwin)
    XARGS:=xargs
endif

TAG:=$(shell git describe --match 'v*' --always --dirty --broken)
LDFLAGS:=-X main.version="${TAG}"

all: build

.PHONY: build
build:
	@nix-build -A default --argstr version "$(TAG)"

# the "debug" build includes a debugger-friendly binary, and fetches the webui
# from the filesystem instead of packing it into the binary
.PHONY: build/debug
build/debug:
	@nix-build -A default --argstr version "$(TAG)" --arg debug true

.PHONY: build/without-nix
build/without-nix:
	@go generate
	@npm run --prefix webui build
	@go run webui/pack_webui.go
	@go build -ldflags "$(LDFLAGS)" .

build/debug/without-nix:
	@go generate
	@npm run --prefix webui build
	@go build -ldflags "$(LDFLAGS)" -gcflags=all="-N -l" .

.PHONY: release
release:
	@nix-build -A release --argstr version "$(TAG)"

.PHONY: secure
secure:
	go run golang.org/x/vuln/cmd/govulncheck ./...

.PHONY: test
test:
	go test -v -bench=. ./...

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
