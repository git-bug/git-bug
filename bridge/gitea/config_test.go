package gitea

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/git-bug/git-bug/bridge/core/auth"
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
			name: "bad url",
			args: args{
				url: "https://gite.com/git-bug/git-bug.git",
			},
			want: want{
				err: ErrBadProjectURL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, owner, project, err := splitURL(tt.args.url)
			assert.Equal(t, tt.want.err, err)
			assert.Equal(t, tt.want.baseURL, baseURL)
			assert.Equal(t, tt.want.owner, owner)
			assert.Equal(t, tt.want.project, project)
		})
	}
}

func TestValidateUsername(t *testing.T) {
	if env := os.Getenv("TRAVIS"); env == "true" {
		t.Skip("Travis environment: avoiding non authenticated requests")
	}
	if _, has := os.LookupEnv("CI"); has {
		t.Skip("Github action environment: avoiding non authenticated requests")
	}

	tests := []struct {
		baseURL string
		name    string
		input   string
		fixed   string
		ok      bool
	}{
		{
			name:    "existing username",
			baseURL: "https://gitea.com/",
			input:   "gitea",
			ok:      true,
		},
		{
			name:    "existing username with bad case",
			baseURL: "https://gitea.com/",
			input:   "GiTeA",
			ok:      true,
		},
		{
			name:    "existing organisation",
			baseURL: "https://gitea.com/",
			input:   "gitea",
			ok:      true,
		},
		{
			name:    "existing organisation with bad case",
			baseURL: "https://gitea.com/",
			input:   "gItEa",
			ok:      true,
		},
		{
			name:    "non existing username",
			baseURL: "https://gitea.com/",
			input:   "cant-find-this",
			ok:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := validateUsername(tt.baseURL, tt.input)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestValidateProject(t *testing.T) {
	envPrivate := os.Getenv("GITEA_TOKEN_PRIVATE")
	if envPrivate == "" {
		t.Skip("Env var GITEA_TOKEN_PRIVATE missing")
	}

	envPublic := os.Getenv("GITEA_TOKEN_PUBLIC")
	if envPublic == "" {
		t.Skip("Env var GITEA_TOKEN_PUBLIC missing")
	}

	tokenPrivate := auth.NewToken(target, envPrivate)
	tokenPublic := auth.NewToken(target, envPublic)

	type args struct {
		baseURL string
		owner   string
		project string
		token   *auth.Token
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "public repository and token with scope 'public_repo'",
			args: args{
				baseURL: "https://gitea.com/",
				project: "gitea",
				owner:   "gitea",
				token:   tokenPublic,
			},
			want: true,
		},
		{
			name: "private repository and token with scope 'repo'",
			args: args{
				baseURL: "https://gitea.com/",
				project: "test-gitea-bridge",
				owner:   "git-bug",
				token:   tokenPrivate,
			},
			want: true,
		},
		{
			name: "private repository and token with scope 'public_repo'",
			args: args{
				baseURL: "https://gitea.com/",
				project: "test-gitea-bridge",
				owner:   "git-bug",
				token:   tokenPublic,
			},
			want: false,
		},
		{
			name: "project not existing",
			args: args{
				baseURL: "https://gitea.com/",
				project: "cant-find-this",
				owner:   "organisation-not-found",
				token:   tokenPublic,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := validateProject(tt.args.baseURL, tt.args.owner, tt.args.project, tt.args.token)
			assert.Equal(t, tt.want, ok)
		})
	}
}
