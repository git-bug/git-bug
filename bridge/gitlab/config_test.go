package gitlab

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectPath(t *testing.T) {
	type args struct {
		url string
	}
	type want struct {
		path string
		err  error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "default url",
			args: args{
				url: "https://gitlab.com/git-bug/git-bug",
			},
			want: want{
				path: "git-bug/git-bug",
				err:  nil,
			},
		},
		{
			name: "multiple sub groups",
			args: args{
				url: "https://gitlab.com/git-bug/group/subgroup/git-bug",
			},
			want: want{
				path: "git-bug/group/subgroup/git-bug",
				err:  nil,
			},
		},
		{
			name: "default url with git extension",
			args: args{
				url: "https://gitlab.com/git-bug/git-bug.git",
			},
			want: want{
				path: "git-bug/git-bug",
				err:  nil,
			},
		},
		{
			name: "url with git protocol",
			args: args{
				url: "git://gitlab.com/git-bug/git-bug.git",
			},
			want: want{
				path: "git-bug/git-bug",
				err:  nil,
			},
		},
		{
			name: "ssh url",
			args: args{
				url: "git@gitlab.com/git-bug/git-bug.git",
			},
			want: want{
				path: "git-bug/git-bug",
				err:  nil,
			},
		},
		{
			name: "bad url",
			args: args{
				url: "---,%gitlab.com/git-bug/git-bug.git",
			},
			want: want{
				err: ErrBadProjectURL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := getProjectPath(defaultBaseURL, tt.args.url)
			assert.Equal(t, tt.want.path, path)
			assert.Equal(t, tt.want.err, err)
		})
	}
}

func TestValidTokenFormat(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{
			name:  "legacy token without prefix",
			token: "token-string-here123",
			want:  true,
		},
		{
			name:  "legacy token with prefix",
			token: "glpat-token-string-here123",
			want:  true,
		},
		{
			// https://github.com/trufflesecurity/trufflehog/issues/4551, as
			// issued by both gitlab.com and a self-managed instance
			name:  "routable token",
			token: "glpat-rQxN4XPLm8wF_jGbHv_A9731YZq2Kfe0BD.01.6z70tqjnm",
			want:  true,
		},
		{
			// the token from the issue report
			name:  "routable token from the issue report",
			token: "glpat-bE8-c60h8AD4A0v-59rAr246MQj8ObF1DA.01.0y0nl1q64",
			want:  true,
		},
		{
			name:  "empty token",
			token: "",
			want:  false,
		},
		{
			name:  "too short",
			token: "glpat-tooshort",
			want:  false,
		},
		{
			// the payload is one character over the 300 maximum
			name:  "routable payload too long",
			token: "glpat-" + strings.Repeat("a", 301) + ".01.6z70tqjnm",
			want:  false,
		},
		{
			name:  "no a token at all",
			token: "aaaaa",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, validateToken(tt.token))
		})
	}
}
