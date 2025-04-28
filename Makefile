UNAME_S := $(shell uname -s)
XARGS:=xargs -r
ifeq ($(UNAME_S),Darwin)
    XARGS:=xargs
endif

TAG:=$(shell git describe --match 'v*' --always --dirty --broken)
LDFLAGS:=-X main.version="${TAG}"

all: build

.PHONY: build-webui
build-webui:
	cd webui && pnpm install && pnpm run build

.PHONY: build
build: build-webui
	go generate
	go build -tags webui -ldflags "$(LDFLAGS)" .

# produce a debugger-friendly build
.PHONY: build/debug
build/debug: build-webui
	go generate
	go build -tags webui -ldflags "$(LDFLAGS)" -gcflags=all="-N -l" .

.PHONY: install
install: build-webui
	go generate
	go install -tags webui -ldflags "$(LDFLAGS)" .

.PHONY: secure
secure: secure-practices secure-vulnerabilities

.PHONY: secure-practices
secure-practices:
	go run github.com/praetorian-inc/gokart scan
	# eventually go run github.com/securego/gosec/v2/cmd/gosec@latest ./...

.PHONY: secure-vulnerabilities
secure-vulnerabilities:
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
