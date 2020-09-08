module github.com/MichaelMure/git-bug

go 1.13

require (
	github.com/99designs/gqlgen v0.10.3-0.20200209012558-b7a58a1c0e4b
	github.com/99designs/keyring v1.1.5
	github.com/MichaelMure/go-term-text v0.2.9
	github.com/araddon/dateparse v0.0.0-20190622164848-0fb0a474d195
	github.com/awesome-gocui/gocui v0.6.1-0.20191115151952-a34ffb055986
	github.com/blang/semver v3.5.1+incompatible
	github.com/cheekybits/genny v0.0.0-20170328200008-9127e812e1e9
	github.com/corpix/uarand v0.1.1 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.9.0
	github.com/go-git/go-git/v5 v5.1.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/icrowley/fake v0.0.0-20180203215853-4178557ae428
	github.com/mattn/go-isatty v0.0.12
	github.com/phayes/freeport v0.0.0-20171002181615-b8543db493a5
	github.com/pkg/errors v0.9.1
	github.com/shurcooL/githubv4 v0.0.0-20190601194912-068505affed7
	github.com/shurcooL/graphql v0.0.0-20181231061246-d48a9a75455f // indirect
	github.com/skratchdot/open-golang v0.0.0-20190402232053-79abb63cd66e
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/vektah/gqlparser v1.3.1
	github.com/xanzy/go-gitlab v0.33.0
	golang.org/x/crypto v0.0.0-20200302210943-78000ba7a073
	golang.org/x/oauth2 v0.0.0-20181106182150-f42d05182288
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/text v0.3.3
)

// Use a forked go-git for now until https://github.com/go-git/go-git/pull/112 is merged
// and released.
replace github.com/go-git/go-git/v5 => github.com/MichaelMure/go-git/v5 v5.1.1-0.20200827115354-b40ca794fe33
