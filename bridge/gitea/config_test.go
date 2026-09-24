package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
)

func TestSplitURL(t *testing.T) {
	type args struct {
		url string
	}
	type want struct {
		baseURL string
		owner   string
		project string
		err     error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "default url",
			args: args{
				url: "https://gitea.com/git-bug/git-bug",
			},
			want: want{
				baseURL: "https://gitea.com/",
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "default issues url",
			args: args{
				url: "https://gitea.com/git-bug/git-bug/issues",
			},
			want: want{
				baseURL: "https://gitea.com/",
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "default url with git extension",
			args: args{
				url: "https://gitea.com/git-bug/git-bug.git",
			},
			want: want{
				baseURL: "https://gitea.com/",
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "url with git protocol",
			args: args{
				url: "git://gitea.com/git-bug/git-bug.git",
			},
			want: want{
				baseURL: "https://gitea.com/",
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "ssh url",
			args: args{
				url: "git@gitea.com:git-bug/git-bug.git",
			},
			want: want{
				baseURL: "https://gitea.com/",
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "issue page url",
			args: args{url: "https://codeberg.org/forgejo/forgejo/issues/123"},
			want: want{baseURL: "https://codeberg.org/", owner: "forgejo", project: "forgejo"},
		},
		{
			name: "plain http with port is kept",
			args: args{url: "http://localhost:3000/owner/repo"},
			want: want{baseURL: "http://localhost:3000/", owner: "owner", project: "repo"},
		},
		{
			name: "instance under a subpath",
			args: args{url: "https://example.com/gitea/owner/repo/pulls"},
			want: want{baseURL: "https://example.com/gitea/", owner: "owner", project: "repo"},
		},
		{
			name: "ssh url with port",
			args: args{url: "ssh://git@gitea.com:2222/git-bug/git-bug.git"},
			want: want{baseURL: "https://gitea.com/", owner: "git-bug", project: "git-bug"},
		},
		{
			name: "owner only",
			args: args{url: "https://gitea.com/git-bug"},
			want: want{err: ErrBadProjectURL},
		},
		{
			name: "unsupported scheme",
			args: args{url: "ftp://gitea.com/git-bug/git-bug"},
			want: want{err: ErrBadProjectURL},
		},
		{
			name: "bad url",
			args: args{
				url: "xxx",
			},
			want: want{
				err: ErrBadProjectURL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, owner, project, err := splitURL(tt.args.url)
			assert.Equal(t, tt.want.err, err, tt.args.url)
			assert.Equal(t, tt.want.baseURL, baseURL, tt.args.url)
			assert.Equal(t, tt.want.owner, owner, tt.args.url)
			assert.Equal(t, tt.want.project, project, tt.args.url)
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		notFound bool
		ok       bool
	}{
		{name: "existing username", input: "alice", ok: true},
		{name: "non existing username", input: "cant-find-this", notFound: true, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
			if tt.notFound {
				fa.NotFoundUsers = []string{tt.input}
			}
			srv := fa.NewServer(t)
			ok, _ := validateUsername(srv.URL, tt.input)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestValidateProject(t *testing.T) {
	token := auth.NewToken(target, "test-token")

	tests := []struct {
		name         string
		repoNotFound bool
		want         bool
	}{
		{name: "existing project", repoNotFound: false, want: true},
		{name: "project not found", repoNotFound: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo", RepoNotFound: tt.repoNotFound}
			srv := fa.NewServer(t)
			ok, _ := validateProject(srv.URL, fa.Owner, fa.Project, token)
			assert.Equal(t, tt.want, ok)
		})
	}
}

func TestValidateProjectServerError(t *testing.T) {
	token := auth.NewToken(target, "test-token")
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo", RepoErrStatus: 500}
	srv := fa.NewServer(t)

	// The current error text is misleading for 5xx responses, so assert only
	// that the server error is not accepted as a valid project.
	ok, err := validateProject(srv.URL, fa.Owner, fa.Project, token)
	assert.False(t, ok, "a 500 from the repo endpoint must not be reported as a valid project")
	assert.Error(t, err, "a 500 from the repo endpoint must surface as a non-nil error")
}
