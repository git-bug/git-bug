package gitlab

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
