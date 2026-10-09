UNAME_S := $(shell uname -s)
XARGS:=xargs -r
ifeq ($(UNAME_S),Darwin)
    XARGS:=xargs
endif

TAG:=$(shell git describe --tags --match 'v*' --always --dirty --broken)
LDFLAGS:=-X main.version="${TAG}"

# Set WEBUI=0 to build without the web UI, which spares you a Node.js/pnpm
# toolchain entirely. The resulting binary simply has none compiled in:
# `git-bug webui` then refuses to run and says so (see //webui:assets_stub.go).
WEBUI?=1
ifeq ($(WEBUI),0)
WEBUI_DEP=
WEBUI_TAG=
else
WEBUI_DEP=build-webui
WEBUI_TAG=-tags webui
endif

all: build

.PHONY: build-webui
build-webui:
	cd webui && pnpm install && pnpm run build

.PHONY: build
build: $(WEBUI_DEP)
	go generate
	go build $(WEBUI_TAG) -ldflags "$(LDFLAGS)" .

# produce a debugger-friendly build
.PHONY: build/debug
build/debug: $(WEBUI_DEP)
	go generate
	go build $(WEBUI_TAG) -ldflags "$(LDFLAGS)" -gcflags=all="-N -l" .

.PHONY: install
install: $(WEBUI_DEP)
	go generate
	go install $(WEBUI_TAG) -ldflags "$(LDFLAGS)" .

.PHONY: secure
secure:
	go tool govulncheck ./...

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

.PHONY: clean-remote-identities
clean-remote-identities:
	git ls-remote origin "refs/identities/*" | cut -f 2 | $(XARGS) git push origin -d
