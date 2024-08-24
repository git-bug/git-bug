package gitlab

import (
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
