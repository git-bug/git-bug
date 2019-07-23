package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNewTitle(t *testing.T) {
	type args struct {
		diff string
	}
	type want struct {
		title string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "addition diff",
			args: args{
				diff: "**first issue** to **first issue{+ edited+}**",
			},
			want: want{
				title: "first issue edited",
			},
		},
		{
			name: "deletion diff",
			args: args{
				diff: "**first issue{- edited-}** to **first issue**",
			},
			want: want{
				title: "first issue",
			},
		},
		{
			name: "mixed diff",
			args: args{
				diff: "**first {-issue-}** to **first {+bug+}**",
			},
			want: want{
				title: "first bug",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := getNewTitle(tt.args.diff)
			assert.Equal(t, tt.want.title, title)
		})
	}
}
