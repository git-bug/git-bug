package launchpad

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitURL(t *testing.T) {
	type args struct {
		url string
	}
	type want struct {
		project string
		err     error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "default project url",
			args: args{
				url: "https://launchpad.net/ubuntu",
			},
			want: want{
				project: "ubuntu",
				err:     nil,
			},
		},
		{
			name: "project bugs url",
			args: args{
				url: "https://bugs.launchpad.net/ubuntu",
			},
			want: want{
				project: "ubuntu",
				err:     nil,
			},
		},
		{
			name: "bad url",
			args: args{
				url: "https://launchpa.net/ubuntu",
			},
			want: want{
				err: ErrBadProjectURL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := splitURL(tt.args.url)
			assert.Equal(t, tt.want.err, err)
			assert.Equal(t, tt.want.project, project)
		})
	}
}

func TestValidateProject(t *testing.T) {
	type args struct {
		project string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "public project",
			args: args{
				project: "ubuntu",
			},
			want: true,
		},
		{
			name: "non existing project",
			args: args{
				project: "cant-find-this",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := validateProject(tt.args.project)
			assert.Equal(t, tt.want, ok)
		})
	}
}
