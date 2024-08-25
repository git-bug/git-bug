package github

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
				url: "https://github.com/git-bug/git-bug",
			},
			want: want{
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "default issues url",
			args: args{
				url: "https://github.com/git-bug/git-bug/issues",
			},
			want: want{
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "default url with git extension",
			args: args{
				url: "https://github.com/git-bug/git-bug.git",
			},
			want: want{
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "url with git protocol",
			args: args{
				url: "git://github.com/git-bug/git-bug.git",
			},
			want: want{
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "ssh url",
			args: args{
				url: "git@github.com:git-bug/git-bug.git",
			},
			want: want{
				owner:   "git-bug",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "bad url",
			args: args{
				url: "https://githb.com/git-bug/git-bug.git",
			},
			want: want{
				err: ErrBadProjectURL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, project, err := splitURL(tt.args.url)
			assert.Equal(t, tt.want.err, err)
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
		name  string
		input string
		fixed string
		ok    bool
	}{
		{
			name:  "existing username",
			input: "git-bug",
			fixed: "git-bug",
			ok:    true,
		},
		{
			name:  "existing username with bad case",
			input: "GiT-bUg",
			fixed: "git-bug",
			ok:    true,
		},
		{
			name:  "existing organisation",
			input: "git-bug",
			fixed: "git-bug",
			ok:    true,
		},
		{
			name:  "existing organisation with bad case",
			input: "gIt-BuG",
			fixed: "git-bug",
			ok:    true,
		},
		{
			name:  "non existing username",
			input: "cant-find-this",
			fixed: "",
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, fixed, err := validateUsername(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.fixed, fixed)
		})
	}
}

func TestValidateProject(t *testing.T) {
	envPrivate := os.Getenv("GITHUB_TOKEN_PRIVATE")
	if envPrivate == "" {
		t.Skip("Env var GITHUB_TOKEN_PRIVATE missing")
	}

	envPublic := os.Getenv("GITHUB_TOKEN_PUBLIC")
	if envPublic == "" {
		t.Skip("Env var GITHUB_TOKEN_PUBLIC missing")
	}

	tokenPrivate := auth.NewToken(target, envPrivate)
	tokenPublic := auth.NewToken(target, envPublic)

	type args struct {
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
				project: "git-bug",
				owner:   "git-bug",
				token:   tokenPublic,
			},
			want: true,
		},
		{
			name: "private repository and token with scope 'repo'",
			args: args{
				project: "test-github-bridge",
				owner:   "git-bug",
				token:   tokenPrivate,
			},
			want: true,
		},
		{
			name: "private repository and token with scope 'public_repo'",
			args: args{
				project: "test-github-bridge",
				owner:   "git-bug",
				token:   tokenPublic,
			},
			want: false,
		},
		{
			name: "project not existing",
			args: args{
				project: "cant-find-this",
				owner:   "organisation-not-found",
				token:   tokenPublic,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := validateProject(tt.args.owner, tt.args.project, tt.args.token)
			assert.Equal(t, tt.want, ok)
		})
	}
}
