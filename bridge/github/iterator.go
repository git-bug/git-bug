package github

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/shurcooL/githubv4"
)

/**
type iterator interface {
	Count() int
	Error() error

	NextIssue() bool
	NextIssueEdit() bool
	NextTimeline() bool
	NextCommentEdit() bool

	IssueValue() issueTimeline
	IssueEditValue() userContentEdit
	TimelineValue() timelineItem
	CommentEditValue() userContentEdit
}
*/

type indexer struct{ index int }

type issueEditIterator struct {
	index     int
	query     issueEditQuery
	variables map[string]interface{}
}

type commentEditIterator struct {
	index     int
	query     commentEditQuery
	variables map[string]interface{}
}

type timelineIterator struct {
	index     int
	query     issueTimelineQuery
	variables map[string]interface{}

	issueEdit   indexer
	commentEdit indexer

	lastEndCursor githubv4.String // storing timeline end cursor for future use
}

type iterator struct {
	// github graphql client
	gc *githubv4.Client

	// if since is given the iterator will query only the updated
	// and created issues after this date
	since time.Time

	// number of timelines/userEditcontent/issueEdit to query
	// at a time more capacity = more used memory = less queries
	// to make
	capacity int

	// sticky error
	err error

	// count to keep track of the number of imported issues
	count int

	// timeline iterator
	timeline timelineIterator

	// issue edit iterator
	issueEdit issueEditIterator

	// comment edit iterator
	commentEdit commentEditIterator
}

func newIterator(conf core.Configuration, since time.Time) *iterator {
	return &iterator{
		since:    since,
		gc:       buildClient(conf),
		capacity: 10,
		count:    0,
		timeline: timelineIterator{
			index:       -1,
			issueEdit:   indexer{-1},
			commentEdit: indexer{-1},
			variables: map[string]interface{}{
				"owner": githubv4.String(conf["user"]),
				"name":  githubv4.String(conf["project"]),
			},
		},
		commentEdit: commentEditIterator{
			index: -1,
			variables: map[string]interface{}{
				"owner": githubv4.String(conf["user"]),
				"name":  githubv4.String(conf["project"]),
			},
		},
		issueEdit: issueEditIterator{
			index: -1,
			variables: map[string]interface{}{
				"owner": githubv4.String(conf["user"]),
				"name":  githubv4.String(conf["project"]),
			},
		},
	}
}

// init issue timeline variables
func (i *iterator) initTimelineQueryVariables() {
	i.timeline.variables["issueFirst"] = githubv4.Int(1)
	i.timeline.variables["issueAfter"] = (*githubv4.String)(nil)
	i.timeline.variables["issueSince"] = githubv4.DateTime{Time: i.since}
	i.timeline.variables["timelineFirst"] = githubv4.Int(i.capacity)
	i.timeline.variables["timelineAfter"] = (*githubv4.String)(nil)
	i.timeline.variables["issueEditLast"] = githubv4.Int(i.capacity)
	i.timeline.variables["issueEditBefore"] = (*githubv4.String)(nil)
	i.timeline.variables["commentEditLast"] = githubv4.Int(i.capacity)
	i.timeline.variables["commentEditBefore"] = (*githubv4.String)(nil)
}

// init issue edit variables
func (i *iterator) initIssueEditQueryVariables() {
	i.issueEdit.variables["issueFirst"] = githubv4.Int(1)
	i.issueEdit.variables["issueAfter"] = i.timeline.variables["issueAfter"]
	i.issueEdit.variables["issueSince"] = githubv4.DateTime{Time: i.since}
	i.issueEdit.variables["issueEditLast"] = githubv4.Int(i.capacity)
	i.issueEdit.variables["issueEditBefore"] = (*githubv4.String)(nil)
}

// init issue comment variables
func (i *iterator) initCommentEditQueryVariables() {
	i.commentEdit.variables["issueFirst"] = githubv4.Int(1)
	i.commentEdit.variables["issueAfter"] = i.timeline.variables["issueAfter"]
	i.commentEdit.variables["issueSince"] = githubv4.DateTime{Time: i.since}
	i.commentEdit.variables["timelineFirst"] = githubv4.Int(1)
	i.commentEdit.variables["timelineAfter"] = (*githubv4.String)(nil)
	i.commentEdit.variables["commentEditLast"] = githubv4.Int(i.capacity)
	i.commentEdit.variables["commentEditBefore"] = (*githubv4.String)(nil)
}

// reverse UserContentEdits arrays in both of the issue and
// comment timelines
func (i *iterator) reverseTimelineEditNodes() {
	reverseEdits(i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes)
	for index, ce := range i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges {
		if ce.Node.Typename == "IssueComment" && len(i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges) != 0 {
			reverseEdits(i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[index].Node.IssueComment.UserContentEdits.Nodes)
		}
	}
}

// Error return last encountered error
func (i *iterator) Error() error {
	return i.err
}

// Count return number of issues we iterated over
func (i *iterator) Count() int {
	return i.count
}

// Next issue
func (i *iterator) NextIssue() bool {
	// we make the first move
	if i.count == 0 {

		// init variables and goto queryIssue block
		i.initTimelineQueryVariables()
		goto queryIssue
	}

	if i.err != nil {
		return false
	}

	if !i.timeline.query.Repository.Issues.PageInfo.HasNextPage {
		return false
	}

	// if we have more issues, query them
	i.timeline.variables["timelineAfter"] = (*githubv4.String)(nil)
	i.timeline.variables["issueAfter"] = i.timeline.query.Repository.Issues.PageInfo.EndCursor
	i.timeline.index = -1

	// store cursor for future use
	i.timeline.lastEndCursor = i.timeline.query.Repository.Issues.Nodes[0].Timeline.PageInfo.EndCursor

	// query issue block
queryIssue:
	if err := i.gc.Query(context.TODO(), &i.timeline.query, i.timeline.variables); err != nil {
		i.err = err
		return false
	}

	if len(i.timeline.query.Repository.Issues.Nodes) == 0 {
		return false
	}

	i.reverseTimelineEditNodes()
	i.count++
	return true
}

func (i *iterator) IssueValue() issueTimeline {
	return i.timeline.query.Repository.Issues.Nodes[0]
}

func (i *iterator) NextTimeline() bool {
	if i.err != nil {
		return false
	}

	if len(i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges) == 0 {
		return false
	}

	if i.timeline.index < min(i.capacity, len(i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges))-1 {
		i.timeline.index++
		return true
	}

	if !i.timeline.query.Repository.Issues.Nodes[0].Timeline.PageInfo.HasNextPage {
		return false
	}

	i.timeline.lastEndCursor = i.timeline.query.Repository.Issues.Nodes[0].Timeline.PageInfo.EndCursor

	// more timelines, query them
	i.timeline.variables["timelineAfter"] = i.timeline.query.Repository.Issues.Nodes[0].Timeline.PageInfo.EndCursor
	if err := i.gc.Query(context.TODO(), &i.timeline.query, i.timeline.variables); err != nil {
		i.err = err
		return false
	}

	i.reverseTimelineEditNodes()
	i.timeline.index = 0
	return true
}

func (i *iterator) TimelineValue() timelineItem {
	return i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index].Node
}

func (i *iterator) NextIssueEdit() bool {
	if i.err != nil {
		return false
	}

	// this mean we looped over all available issue edits in the timeline.
	// now we have to use i.issueEditQuery
	if i.timeline.issueEdit.index == -2 {
		if i.issueEdit.index < min(i.capacity, len(i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes))-1 {
			i.issueEdit.index++
			return true
		}

		if !i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.PageInfo.HasPreviousPage {
			i.timeline.issueEdit.index = -1
			i.issueEdit.index = -1
			return false
		}

		// if there is more edits, query them
		i.issueEdit.variables["issueEditBefore"] = i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.PageInfo.StartCursor
		goto queryIssueEdit
	}

	// if there is no edits
	if len(i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes) == 0 {
		return false
	}

	// loop over them timeline comment edits
	if i.timeline.issueEdit.index < min(i.capacity, len(i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes))-1 {
		i.timeline.issueEdit.index++
		return true
	}

	if !i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.PageInfo.HasPreviousPage {
		i.timeline.issueEdit.index = -1
		return false
	}

	// if there is more edits, query them
	i.initIssueEditQueryVariables()
	i.issueEdit.variables["issueEditBefore"] = i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.PageInfo.StartCursor

queryIssueEdit:
	if err := i.gc.Query(context.TODO(), &i.issueEdit.query, i.issueEdit.variables); err != nil {
		i.err = err
		//i.timeline.issueEdit.index = -1
		return false
	}

	// reverse issue edits because github
	reverseEdits(i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes)

	// this is not supposed to happen
	if len(i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes) == 0 {
		i.timeline.issueEdit.index = -1
		return false
	}

	i.issueEdit.index = 0
	i.timeline.issueEdit.index = -2
	return true
}

func (i *iterator) IssueEditValue() userContentEdit {
	// if we are using issue edit query
	if i.timeline.issueEdit.index == -2 {
		return i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes[i.issueEdit.index]
	}

	// else get it from timeline issue edit query
	return i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes[i.timeline.issueEdit.index]
}

func (i *iterator) NextCommentEdit() bool {
	if i.err != nil {
		return false
	}

	// same as NextIssueEdit
	if i.timeline.commentEdit.index == -2 {

		if i.commentEdit.index < min(i.capacity, len(i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.Nodes))-1 {
			i.commentEdit.index++
			return true
		}

		if !i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.PageInfo.HasPreviousPage {
			i.timeline.commentEdit.index = -1
			i.commentEdit.index = -1
			return false
		}

		// if there is more comment edits, query them
		i.commentEdit.variables["commentEditBefore"] = i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.PageInfo.StartCursor
		goto queryCommentEdit
	}

	// if there is no comment edits
	if len(i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index].Node.IssueComment.UserContentEdits.Nodes) == 0 {
		return false
	}

	// loop over them timeline comment edits
	if i.timeline.commentEdit.index < min(i.capacity, len(i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index].Node.IssueComment.UserContentEdits.Nodes))-1 {
		i.timeline.commentEdit.index++
		return true
	}

	if !i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index].Node.IssueComment.UserContentEdits.PageInfo.HasPreviousPage {
		i.timeline.commentEdit.index = -1
		return false
	}

	i.initCommentEditQueryVariables()
	if i.timeline.index == 0 {
		i.commentEdit.variables["timelineAfter"] = i.timeline.lastEndCursor
	} else {
		i.commentEdit.variables["timelineAfter"] = i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index-1].Cursor
	}

	i.commentEdit.variables["commentEditBefore"] = i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index].Node.IssueComment.UserContentEdits.PageInfo.StartCursor

queryCommentEdit:
	if err := i.gc.Query(context.TODO(), &i.commentEdit.query, i.commentEdit.variables); err != nil {
		i.err = err
		return false
	}

	// this is not supposed to happen
	if len(i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.Nodes) == 0 {
		i.timeline.commentEdit.index = -1
		return false
	}

	reverseEdits(i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.Nodes)

	i.commentEdit.index = 0
	i.timeline.commentEdit.index = -2
	return true
}

func (i *iterator) CommentEditValue() userContentEdit {
	if i.timeline.commentEdit.index == -2 {
		return i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.Nodes[i.commentEdit.index]
	}

	return i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index].Node.IssueComment.UserContentEdits.Nodes[i.timeline.commentEdit.index]
}

func min(a, b int) int {
	if a > b {
		return b
	}

	return a
}
