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
				url: "https://gitlab.com/MichaelMure/git-bug",
			},
			want: want{
				path: "MichaelMure/git-bug",
				err:  nil,
			},
		},
		{
			name: "multiple sub groups",
			args: args{
				url: "https://gitlab.com/MichaelMure/group/subgroup/git-bug",
			},
			want: want{
				path: "MichaelMure/group/subgroup/git-bug",
				err:  nil,
			},
		},
		{
			name: "default url with git extension",
			args: args{
				url: "https://gitlab.com/MichaelMure/git-bug.git",
			},
			want: want{
				path: "MichaelMure/git-bug",
				err:  nil,
			},
		},
		{
			name: "url with git protocol",
			args: args{
				url: "git://gitlab.com/MichaelMure/git-bug.git",
			},
			want: want{
				path: "MichaelMure/git-bug",
				err:  nil,
			},
		},
		{
			name: "ssh url",
			args: args{
				url: "git@gitlab.com/MichaelMure/git-bug.git",
			},
			want: want{
				path: "MichaelMure/git-bug",
				err:  nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := getProjectPath(tt.args.url)
			assert.Equal(t, tt.want.path, path)
			assert.Equal(t, tt.want.err, err)
		})
	}
}
