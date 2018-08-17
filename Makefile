all: build

build:
	go generate
	go build -tags=deploy_build .

install:
	go generate
	go install -tags=deploy_build .

test: build
	go test ./...

pack-webui:
	npm run --prefix webui build
	go run webui/pack_webui.go

clean-local-bugs:
	git for-each-ref refs/bugs/ | cut -f 2 | xargs -r -n 1 git update-ref -d
	git for-each-ref refs/remotes/origin/bugs/ | cut -f 2 | xargs -r -n 1 git update-ref -d

clean-remote-bugs:
	git ls-remote origin "refs/bugs/*" | cut -f 2 | xargs -r git push origin -d

.PHONY: build install test pack-webui clean-local-bugs clean-remote-bugs
