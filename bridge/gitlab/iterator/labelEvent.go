package iterator

import (
	"context"
	"sort"

	"github.com/xanzy/go-gitlab"
)

// Since Gitlab does not return the label events items in the correct order
// we need to sort the list ourselves and stop relying on the pagination model
// #BecauseGitlab
type labelEventIterator struct {
	issue int
	index int
	cache []*gitlab.LabelEvent
}

func newLabelEventIterator() *labelEventIterator {
	lei := &labelEventIterator{}
	lei.Reset(-1)
	return lei
}

func (lei *labelEventIterator) Next(ctx context.Context, conf config) (bool, error) {
	// first query
	if lei.cache == nil {
		return lei.getNext(ctx, conf)
	}

	// move cursor index
	if lei.index < len(lei.cache)-1 {
		lei.index++
		return true, nil
	}

	return false, nil
}

func (lei *labelEventIterator) Value() *gitlab.LabelEvent {
	return lei.cache[lei.index]
}

func (lei *labelEventIterator) getNext(ctx context.Context, conf config) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, conf.timeout)
	defer cancel()

	// since order is not guaranteed we should query all label events
	// and sort them by ID
	page := 1
	for {
		labelEvents, resp, err := conf.gc.ResourceLabelEvents.ListIssueLabelEvents(
			conf.project,
			lei.issue,
			&gitlab.ListLabelEventsOptions{
				ListOptions: gitlab.ListOptions{
					Page:    page,
					PerPage: conf.capacity,
				},
			},
			gitlab.WithContext(ctx),
		)
		if err != nil {
			lei.Reset(-1)
			return false, err
		}

		if len(labelEvents) == 0 {
			break
		}

		lei.cache = append(lei.cache, labelEvents...)

		if resp.TotalPages == page {
			break
		}

		page++
	}

	sort.Sort(lei)
	lei.index = 0

	return len(lei.cache) > 0, nil
}

func (lei *labelEventIterator) Reset(issue int) {
	lei.issue = issue
	lei.index = -1
	lei.cache = nil
}

// ORDERING

func (lei *labelEventIterator) Len() int {
	return len(lei.cache)
}

func (lei *labelEventIterator) Swap(i, j int) {
	lei.cache[i], lei.cache[j] = lei.cache[j], lei.cache[i]
}

func (lei *labelEventIterator) Less(i, j int) bool {
	return lei.cache[i].ID < lei.cache[j].ID
}
