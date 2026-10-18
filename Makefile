UNAME_S := $(shell uname -s)
XARGS:=xargs -r
ifeq ($(UNAME_S),Darwin)
    XARGS:=xargs
endif

TAG:=$(shell git describe --tags --match 'v*' --always --dirty --broken)
LDFLAGS:=-X main.version="${TAG}"

all: build

.PHONY: webui/dist
webui/dist:
	cd webui && pnpm install && pnpm run build

.PHONY: build
build: webui/dist
	go generate
	go build -tags webui -ldflags "$(LDFLAGS)" .

.PHONY: install
install: webui/dist
	go generate
	go install -tags webui -ldflags "$(LDFLAGS)" .

# build or install without the web UI, which doesn't require node and pnpm
.PHONY: build/noweb
build/noweb:
	go generate
	go build -ldflags "$(LDFLAGS)" .

.PHONY: install/noweb
install/noweb:
	go generate
	go install -ldflags "$(LDFLAGS)" .

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
