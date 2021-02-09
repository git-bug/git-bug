package iterator

import (
	"context"
	"sort"

	"github.com/xanzy/go-gitlab"
)

type stateEventIterator struct {
	issue int
	index int
	cache []*gitlab.StateEvent
}

func newStateEventIterator() *stateEventIterator {
	sei := &stateEventIterator{}
	sei.Reset(-1)
	return sei
}

func (sei *stateEventIterator) Next(ctx context.Context, conf config) (bool, error) {
	// first query
	if sei.cache == nil {
		return sei.getNext(ctx, conf)
	}

	// move cursor index
	if sei.index < len(sei.cache)-1 {
		sei.index++
		return true, nil
	}

	return false, nil
}

func (sei *stateEventIterator) Value() *gitlab.StateEvent {
	return sei.cache[sei.index]
}

func (sei *stateEventIterator) getNext(ctx context.Context, conf config) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, conf.timeout)
	defer cancel()

	// since order is not guaranteed we should query all state events
	// and sort them by ID
	page := 1
	for {
		stateEvents, resp, err := conf.gc.ResourceStateEvents.ListIssueStateEvents(
			conf.project,
			sei.issue,
			&gitlab.ListStateEventsOptions{
				ListOptions: gitlab.ListOptions{
					Page:    page,
					PerPage: conf.capacity,
				},
			},
			gitlab.WithContext(ctx),
		)
		if err != nil {
			sei.Reset(-1)
			return false, err
		}

		if len(stateEvents) == 0 {
			break
		}

		sei.cache = append(sei.cache, stateEvents...)

		if resp.TotalPages == page {
			break
		}

		page++
	}

	sort.Sort(sei)
	sei.index = 0

	return len(sei.cache) > 0, nil
}

func (sei *stateEventIterator) Reset(issue int) {
	sei.issue = issue
	sei.index = -1
	sei.cache = nil
}

// ORDERING

func (sei *stateEventIterator) Len() int {
	return len(sei.cache)
}

func (sei *stateEventIterator) Swap(i, j int) {
	sei.cache[i], sei.cache[j] = sei.cache[j], sei.cache[i]
}

func (sei *stateEventIterator) Less(i, j int) bool {
	return sei.cache[i].ID < sei.cache[j].ID
}
