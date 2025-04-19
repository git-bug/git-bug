package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTitleParser(t *testing.T) {
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
			name: "simple addition (html)",
			args: args{
				diff: `<p>changed title from <code class="idiff">simple title</code> to <code class="idiff">simple title<span class="idiff left addition"> addition</span></code></p>`,
			},
			want: want{
				title: "simple title addition",
			},
		},
		{
			name: "simple addition (markdown)",
			args: args{
				diff: `changed title from **simple title** to **simple title{+ addition+}**`,
			},
			want: want{
				title: "simple title addition",
			},
		},
		{
			name: "simple deletion (html)",
			args: args{
				diff: `<p>changed title from <code class="idiff">simple<span class="idiff left right deletion"> deleted</span> title</code> to <code class="idiff">simple title</code></p>`,
			},
			want: want{
				title: "simple title",
			},
		},
		{
			name: "simple deletion (markdown)",
			args: args{
				diff: `changed title from **simple{- deleted-} title** to **simple title**`,
			},
			want: want{
				title: "simple title",
			},
		},
		{
			name: "tail replacement (html)",
			args: args{
				diff: `<p>changed title from <code class="idiff">tail <span class="idiff left right deletion">title</span></code> to <code class="idiff">tail <span class="idiff left addition">replacement</span></code></p>`,
			},
			want: want{
				title: "tail replacement",
			},
		},
		{
			name: "tail replacement (markdown)",
			args: args{
				diff: `changed title from **tail {-title-}** to **tail {+replacement+}**`,
			},
			want: want{
				title: "tail replacement",
			},
		},
		{
			name: "head replacement (html)",
			args: args{
				diff: `<p>changed title from <code class="idiff"><span class="idiff left right deletion">title</span> replacement</code> to <code class="idiff"><span class="idiff left addition">head</span> replacement</code></p>`,
			},
			want: want{
				title: "head replacement",
			},
		},
		{
			name: "head replacement (markdown)",
			args: args{
				diff: `changed title from **{-title-} replacement** to **{+head+} replacement**`,
			},
			want: want{
				title: "head replacement",
			},
		},
		{
			name: "complex multi-section diff (html)",
			args: args{
				diff: `<p>changed title from <code class="idiff">this <span class="idiff left right deletion">is</span> an <span class="idiff left right deletion">issue</span></code> to <code class="idiff">this <span class="idiff left addition">may be</span> an <span class="idiff left right addition">amazing bug</span></code></p>`,
			},
			want: want{
				title: "this may be an amazing bug",
			},
		},
		{
			name: "complex multi-section diff (markdown)",
			args: args{
				diff: `changed title from **this {-is-} an {-issue-}** to **this {+may be+} an {+amazing bug+}**`,
			},
			want: want{
				title: "this may be an amazing bug",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, err := NewWithInput(TitleParser, tt.args.diff).Parse()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, tt.want.title, title)
		})
	}
}
