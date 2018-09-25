all: build

build:
	go generate
	go build .

# produce a build debugger friendly
debug-build:
	go generate
	go build -gcflags=all="-N -l" .

install:
	go generate
	go install .

test:
	go test -bench=. ./...

pack-webui:
	npm run --prefix webui build
	go run webui/pack_webui.go

# produce a build that will fetch the web UI from the filesystem instead of from the binary
debug-webui:
	go build -tags=debugwebui

clean-local-bugs:
	git for-each-ref refs/bugs/ | cut -f 2 | xargs -r -n 1 git update-ref -d
	git for-each-ref refs/remotes/origin/bugs/ | cut -f 2 | xargs -r -n 1 git update-ref -d

clean-remote-bugs:
	git ls-remote origin "refs/bugs/*" | cut -f 2 | xargs -r git push origin -d

.PHONY: build install test pack-webui debug-webui clean-local-bugs clean-remote-bugs
