package github

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
				url: "https://github.com/MichaelMure/git-bug",
			},
			want: want{
				owner:   "MichaelMure",
				project: "git-bug",
				err:     nil,
			},
		},

		{
			name: "default url with git extension",
			args: args{
				url: "https://github.com/MichaelMure/git-bug.git",
			},
			want: want{
				owner:   "MichaelMure",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "url with git protocol",
			args: args{
				url: "git://github.com/MichaelMure/git-bug.git",
			},
			want: want{
				owner:   "MichaelMure",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "ssh url",
			args: args{
				url: "git@github.com:MichaelMure/git-bug.git",
			},
			want: want{
				owner:   "MichaelMure",
				project: "git-bug",
				err:     nil,
			},
		},
		{
			name: "bad url",
			args: args{
				url: "https://githb.com/MichaelMure/git-bug.git",
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

func TestValidateProject(t *testing.T) {
	tokenPrivateScope := os.Getenv("GITHUB_TOKEN_PRIVATE")
	if tokenPrivateScope == "" {
		t.Skip("Env var GITHUB_TOKEN_PRIVATE missing")
	}

	tokenPublicScope := os.Getenv("GITHUB_TOKEN_PUBLIC")
	if tokenPublicScope == "" {
		t.Skip("Env var GITHUB_TOKEN_PUBLIC missing")
	}

	type args struct {
		owner   string
		project string
		token   string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "public repository and token with scope 'public_repo",
			args: args{
				project: "git-bug",
				owner:   "MichaelMure",
				token:   tokenPublicScope,
			},
			want: true,
		},
		{
			name: "private repository and token with scope 'repo",
			args: args{
				project: "git-bug-test-github-bridge",
				owner:   "MichaelMure",
				token:   tokenPrivateScope,
			},
			want: true,
		},
		{
			name: "private repository and token with scope 'public_repo'",
			args: args{
				project: "git-bug-test-github-bridge",
				owner:   "MichaelMure",
				token:   tokenPublicScope,
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
