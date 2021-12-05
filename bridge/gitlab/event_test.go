package gitlab

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

var _ Event = mockEvent(0)

type mockEvent int64

func (m mockEvent) ID() string           { panic("implement me") }
func (m mockEvent) UserID() int          { panic("implement me") }
func (m mockEvent) Kind() EventKind      { panic("implement me") }
func (m mockEvent) CreatedAt() time.Time { return time.Unix(int64(m), 0) }

func TestSortedEvents(t *testing.T) {
	makeInput := func(times ...int64) chan Event {
		out := make(chan Event)
		go func() {
			for _, t := range times {
				out <- mockEvent(t)
			}
			close(out)
		}()
		return out
	}

	sorted := SortedEvents(
		makeInput(),
		makeInput(1, 7, 9, 19),
		makeInput(2, 8, 23),
		makeInput(35, 48, 59, 64, 721),
	)

	var previous Event
	for event := range sorted {
		if previous != nil {
			require.True(t, previous.CreatedAt().Before(event.CreatedAt()))
		}
		previous = event
	}
}
