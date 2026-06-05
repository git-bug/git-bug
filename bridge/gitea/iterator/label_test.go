package iterator

import (
	"context"
	"testing"
	"time"

	gitea "gitea.dev/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
)

func TestLabelIteratorErrorOnMissingTotalCountHeader(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		}},
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {{ID: 1, Type: "label", Created: ts, Label: []*gitea.Label{{ID: 1, Name: "bug"}}}},
		},
		TimelineOmitTotalCount: true,
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())

	require.NotPanics(t, func() {
		assert.False(t, iter.NextLabel())
	})
	assert.Error(t, iter.Error())
}

func TestLabelIteratorReturnsAPIError(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:           "owner",
		Project:         "repo",
		TimelineErrPage: 1,
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		}},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())

	require.NotPanics(t, func() {
		assert.False(t, iter.NextLabel())
	})
	assert.Error(t, iter.Error())
}

// TestLabelIteratorRespectsSince verifies that the iterator only returns label
// events that occurred at or after the since cutoff, using the timeline API.
// The FakeAPI exposes the timeline endpoint with since filtering; the iterator
// should use it rather than the snapshot GetIssueLabels endpoint.
func TestLabelIteratorRespectsSince(t *testing.T) {
	since := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	before := since.Add(-24 * time.Hour)
	after := since.Add(24 * time.Hour)

	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts,
		}},
		// Labels snapshot (what GetIssueLabels returns — both labels, no since filtering).
		Labels: []*gitea.Label{
			{ID: 1, Name: "old-label"},
			{ID: 2, Name: "new-label"},
		},
		// Timeline events with timestamps (what the iterator should use).
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {
				{ID: 1, Type: "label", Created: before, Label: []*gitea.Label{{ID: 1, Name: "old-label"}}},
				{ID: 2, Type: "label", Created: after, Label: []*gitea.Label{{ID: 2, Name: "new-label"}}},
			},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, since)
	require.True(t, iter.NextIssue())

	var names []string
	for iter.NextLabel() {
		names = append(names, iter.LabelValue().Label.Name)
	}
	require.NoError(t, iter.Error())
	assert.Equal(t, []string{"new-label"}, names, "label events before since should be excluded")
}

func TestLabelIteratorOneRequestPerIssue(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "first", Poster: &gitea.User{UserName: "u"}, Created: ts},
			{ID: 2, Index: 2, Title: "second", Poster: &gitea.User{UserName: "u"}, Created: ts},
		},
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {{ID: 1, Type: "label", Created: ts, Label: []*gitea.Label{{ID: 1, Name: "bug"}}}},
			2: {{ID: 2, Type: "label", Created: ts, Label: []*gitea.Label{{ID: 2, Name: "enhancement"}}}},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	for iter.NextIssue() {
		for iter.NextLabel() {
		}
	}
	require.NoError(t, iter.Error())

	assert.Len(t, fa.TimelineRequests, 2,
		"labels are returned in one response per issue; no extra requests expected")
}

// TestLabelIteratorSkipsNonLabelEvents verifies that pages containing only
// non-label timeline events do not stop iteration early.
// The bug: pageIterator stops when fetchLabels returns 0 items, even if
// more=true, so label events on later pages are never fetched.
func TestLabelIteratorSkipsNonLabelEvents(t *testing.T) {
	const capacity = 2
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts,
		}},
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {
				// Page 1: two non-label events (fills the page)
				{ID: 1, Type: "comment", Created: ts},
				{ID: 2, Type: "comment", Created: ts},
				// Page 2: the label event
				{ID: 3, Type: "label", Created: ts, Label: []*gitea.Label{{ID: 1, Name: "bug"}}},
			},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())

	var names []string
	for iter.NextLabel() {
		names = append(names, iter.LabelValue().Label.Name)
	}
	require.NoError(t, iter.Error())
	require.GreaterOrEqual(t, len(fa.TimelineRequests), 2,
		"must request at least 2 pages; if only 1, the bug is early stop on empty label page, not a pagination miss")
	assert.Equal(t, []string{"bug"}, names, "label event on page 2 should not be missed when page 1 has only non-label events")
}

func TestLabelIteratorReturnsLabels(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		}},
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {
				{ID: 1, Type: "label", Created: ts, Label: []*gitea.Label{{ID: 1, Name: "bug"}}},
				{ID: 2, Type: "label", Created: ts, Label: []*gitea.Label{{ID: 2, Name: "enhancement"}}},
			},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())

	var names []string
	for iter.NextLabel() {
		names = append(names, iter.LabelValue().Label.Name)
	}
	require.NoError(t, iter.Error())
	assert.Equal(t, []string{"bug", "enhancement"}, names)
}
