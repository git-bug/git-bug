UNAME_S := $(shell uname -s)
XARGS:=xargs -r
ifeq ($(UNAME_S),Darwin)
    XARGS:=xargs
endif

SYSTEM=$(shell nix eval --impure --expr 'builtins.currentSystem' --raw 2>/dev/null || echo '')

TAG:=$(shell git describe --match 'v*' --always --dirty --broken)
LDFLAGS:=-X main.version="${TAG}"

all: build

.PHONY: list-checks
list-checks:
	@if test -z "$(SYSTEM)"; then echo "unable to detect system. is nix installed?" && exit 1; fi
	@printf "Available checks for $(SYSTEM) (run all with \`nix flake check\`):\n"
	@nix flake show --json 2>/dev/null |\
		dasel -r json -w plain '.checks.x86_64-linux.keys().all()' |\
		xargs -I NAME printf '\t%-20s %s\n' "NAME" "nix build .#checks.linux.NAME"

.PHONY: build-webui
build-webui:
	cd webui && pnpm install && pnpm run build

.PHONY: build
build: build-webui
	go generate
	go build -ldflags "$(LDFLAGS)" .

# produce a debugger-friendly build
.PHONY: build/debug
build/debug: build-webui
	go generate
	go build -ldflags "$(LDFLAGS)" -gcflags=all="-N -l" .

.PHONY: install
install: build-webui
	go generate
	go install -ldflags "$(LDFLAGS)" .

.PHONY: releases
releases: build-webui
	go generate
	go run github.com/mitchellh/gox@v1.0.1 -ldflags "$(LDFLAGS)" -osarch '!darwin/386' -output "dist/{{.Dir}}_{{.OS}}_{{.Arch}}"

secure: secure-practices secure-vulnerabilities

.PHONY: secure-practices
secure-practices:
	go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
	# eventually go run github.com/praetorian-inc/gokart scan

.PHONY: secure-vulnerabilities
secure-vulnerabilities:
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
