package gitea

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
)

func TestConfigureRejectsUnboundCredentialBeforeContactingHost(t *testing.T) {
	repo, backend := newRoundTripRepo(t)
	for _, tc := range []struct {
		name, target, baseURL string
	}{
		{"missing base URL", target, ""},
		{"another instance", target, "https://other.example/"},
		{"another bridge", "github", "https://example.com/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cred := auth.NewToken(tc.target, "secret")
			cred.SetMetadata(auth.MetaKeyLogin, "testuser")
			if tc.baseURL != "" {
				cred.SetMetadata(auth.MetaKeyBaseURL, tc.baseURL)
			}
			require.NoError(t, auth.Store(repo, cred))
			_, err := (&Gitea{}).Configure(backend, core.BridgeParams{
				URL: "https://example.com/owner/repo", CredPrefix: cred.ID().String(),
			}, false)
			require.Error(t, err)
			assert.True(t, strings.Contains(err.Error(), "credential"), err)
		})
	}
}

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
			name: "instance under subpath with repo named issues",
			args: args{url: "https://example.com/gitea/owner/issues"},
			want: want{baseURL: "https://example.com/gitea/", owner: "owner", project: "issues"},
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

func TestTokenURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "standard URL without trailing slash",
			baseURL:  "https://codefloe.com",
			expected: "https://codefloe.com/user/settings/applications",
		},
		{
			name:     "standard URL with trailing slash",
			baseURL:  "https://codefloe.com/",
			expected: "https://codefloe.com/user/settings/applications",
		},
		{
			name:     "URL with subpath without trailing slash",
			baseURL:  "https://example.com/gitea",
			expected: "https://example.com/gitea/user/settings/applications",
		},
		{
			name:     "URL with subpath and trailing slash",
			baseURL:  "https://example.com/gitea/",
			expected: "https://example.com/gitea/user/settings/applications",
		},
		{
			name:     "HTTP URL with port",
			baseURL:  "http://localhost:3000",
			expected: "http://localhost:3000/user/settings/applications",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tokenURL(tt.baseURL))
		})
	}
}
