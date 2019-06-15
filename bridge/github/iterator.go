package github

import (
	"context"
	"time"

	"github.com/shurcooL/githubv4"
)

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

	// lastEndCursor cache the timeline end cursor for one iteration
	lastEndCursor githubv4.String
}

type iterator struct {
	// github graphql client
	gc *githubv4.Client

	// if since is given the iterator will query only the updated
	// and created issues after this date
	since time.Time

	// number of timelines/userEditcontent/issueEdit to query
	// at a time, more capacity = more used memory = less queries
	// to make
	capacity int

	// sticky error
	err error

	// timeline iterator
	timeline timelineIterator

	// issue edit iterator
	issueEdit issueEditIterator

	// comment edit iterator
	commentEdit commentEditIterator
}

// NewIterator create and initialize a new iterator
func NewIterator(owner, project, token string, since time.Time) *iterator {
	i := &iterator{
		gc:       buildClient(token),
		since:    since,
		capacity: 10,
		timeline: timelineIterator{
			index:       -1,
			issueEdit:   indexer{-1},
			commentEdit: indexer{-1},
			variables: map[string]interface{}{
				"owner": githubv4.String(owner),
				"name":  githubv4.String(project),
			},
		},
		commentEdit: commentEditIterator{
			index: -1,
			variables: map[string]interface{}{
				"owner": githubv4.String(owner),
				"name":  githubv4.String(project),
			},
		},
		issueEdit: issueEditIterator{
			index: -1,
			variables: map[string]interface{}{
				"owner": githubv4.String(owner),
				"name":  githubv4.String(project),
			},
		},
	}

	i.initTimelineQueryVariables()
	return i
}

// init issue timeline variables
func (i *iterator) initTimelineQueryVariables() {
	i.timeline.variables["issueFirst"] = githubv4.Int(1)
	i.timeline.variables["issueAfter"] = (*githubv4.String)(nil)
	i.timeline.variables["issueSince"] = githubv4.DateTime{Time: i.since}
	i.timeline.variables["timelineFirst"] = githubv4.Int(i.capacity)
	i.timeline.variables["timelineAfter"] = (*githubv4.String)(nil)
	// Fun fact, github provide the comment edition in reverse chronological
	// order, because haha. Look at me, I'm dying of laughter.
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
	node := i.timeline.query.Repository.Issues.Nodes[0]
	reverseEdits(node.UserContentEdits.Nodes)
	for index, ce := range node.Timeline.Edges {
		if ce.Node.Typename == "IssueComment" && len(node.Timeline.Edges) != 0 {
			reverseEdits(node.Timeline.Edges[index].Node.IssueComment.UserContentEdits.Nodes)
		}
	}
}

// Error return last encountered error
func (i *iterator) Error() error {
	return i.err
}

func (i *iterator) queryIssue() bool {
	if err := i.gc.Query(context.TODO(), &i.timeline.query, i.timeline.variables); err != nil {
		i.err = err
		return false
	}

	if len(i.timeline.query.Repository.Issues.Nodes) == 0 {
		return false
	}

	i.reverseTimelineEditNodes()
	return true
}

// NextIssue try to query the next issue and return true. Only one issue is
// queried at each call.
func (i *iterator) NextIssue() bool {
	// if $issueAfter variable is nil we can directly make the first query
	if i.timeline.variables["issueAfter"] == (*githubv4.String)(nil) {
		nextIssue := i.queryIssue()
		// prevent from infinite loop by setting a non nil cursor
		i.timeline.variables["issueAfter"] = i.timeline.query.Repository.Issues.PageInfo.EndCursor
		return nextIssue
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
	return i.queryIssue()
}

// IssueValue return the actual issue value
func (i *iterator) IssueValue() issueTimeline {
	return i.timeline.query.Repository.Issues.Nodes[0]
}

// NextTimelineItem return true if there is a next timeline item and increments the index by one.
// It is used iterates over all the timeline items. Extra queries are made if it is necessary.
func (i *iterator) NextTimelineItem() bool {
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

// TimelineItemValue return the actual timeline item value
func (i *iterator) TimelineItemValue() timelineItem {
	return i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index].Node
}

func (i *iterator) queryIssueEdit() bool {
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
	return i.nextValidIssueEdit()
}

func (i *iterator) nextValidIssueEdit() bool {
	// issueEdit.Diff == nil happen if the event is older than early 2018, Github doesn't have the data before that.
	// Best we can do is to ignore the event.
	if issueEdit := i.IssueEditValue(); issueEdit.Diff == nil || string(*issueEdit.Diff) == "" {
		return i.NextIssueEdit()
	}
	return true
}

// NextIssueEdit return true if there is a next issue edit and increments the index by one.
// It is used iterates over all the issue edits. Extra queries are made if it is necessary.
func (i *iterator) NextIssueEdit() bool {
	if i.err != nil {
		return false
	}

	// this mean we looped over all available issue edits in the timeline.
	// now we have to use i.issueEditQuery
	if i.timeline.issueEdit.index == -2 {
		if i.issueEdit.index < min(i.capacity, len(i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes))-1 {
			i.issueEdit.index++
			return i.nextValidIssueEdit()
		}

		if !i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.PageInfo.HasPreviousPage {
			i.timeline.issueEdit.index = -1
			i.issueEdit.index = -1
			return false
		}

		// if there is more edits, query them
		i.issueEdit.variables["issueEditBefore"] = i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.PageInfo.StartCursor
		return i.queryIssueEdit()
	}

	// if there is no edit, the UserContentEdits given by github is empty. That
	// means that the original message is given by the issue message.
	//
	// if there is edits, the UserContentEdits given by github contains both the
	// original message and the following edits. The issue message give the last
	// version so we don't care about that.
	//
	// the tricky part: for an issue older than the UserContentEdits API, github
	// doesn't have the previous message version anymore and give an edition
	// with .Diff == nil. We have to filter them.
	if len(i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes) == 0 {
		return false
	}

	// loop over them timeline comment edits
	if i.timeline.issueEdit.index < min(i.capacity, len(i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes))-1 {
		i.timeline.issueEdit.index++
		return i.nextValidIssueEdit()
	}

	if !i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.PageInfo.HasPreviousPage {
		i.timeline.issueEdit.index = -1
		return false
	}

	// if there is more edits, query them
	i.initIssueEditQueryVariables()
	i.issueEdit.variables["issueEditBefore"] = i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.PageInfo.StartCursor
	return i.queryIssueEdit()
}

// IssueEditValue return the actual issue edit value
func (i *iterator) IssueEditValue() userContentEdit {
	// if we are using issue edit query
	if i.timeline.issueEdit.index == -2 {
		return i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes[i.issueEdit.index]
	}

	// else get it from timeline issue edit query
	return i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits.Nodes[i.timeline.issueEdit.index]
}

func (i *iterator) queryCommentEdit() bool {
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
	return i.nextValidCommentEdit()
}

func (i *iterator) nextValidCommentEdit() bool {
	// if comment edit diff is a nil pointer or points to an empty string look for next value
	if commentEdit := i.CommentEditValue(); commentEdit.Diff == nil || string(*commentEdit.Diff) == "" {
		return i.NextCommentEdit()
	}
	return true
}

// NextCommentEdit return true if there is a next comment edit and increments the index by one.
// It is used iterates over all the comment edits. Extra queries are made if it is necessary.
func (i *iterator) NextCommentEdit() bool {
	if i.err != nil {
		return false
	}

	// same as NextIssueEdit
	if i.timeline.commentEdit.index == -2 {

		if i.commentEdit.index < min(i.capacity, len(i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.Nodes))-1 {
			i.commentEdit.index++
			return i.nextValidCommentEdit()
		}

		if !i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.PageInfo.HasPreviousPage {
			i.timeline.commentEdit.index = -1
			i.commentEdit.index = -1
			return false
		}

		// if there is more comment edits, query them
		i.commentEdit.variables["commentEditBefore"] = i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.PageInfo.StartCursor
		return i.queryCommentEdit()
	}

	// if there is no comment edits
	if len(i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index].Node.IssueComment.UserContentEdits.Nodes) == 0 {
		return false
	}

	// loop over them timeline comment edits
	if i.timeline.commentEdit.index < min(i.capacity, len(i.timeline.query.Repository.Issues.Nodes[0].Timeline.Edges[i.timeline.index].Node.IssueComment.UserContentEdits.Nodes))-1 {
		i.timeline.commentEdit.index++
		return i.nextValidCommentEdit()
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

	return i.queryCommentEdit()
}

// CommentEditValue return the actual comment edit value
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

func reverseEdits(edits []userContentEdit) {
	for i, j := 0, len(edits)-1; i < j; i, j = i+1, j-1 {
		edits[i], edits[j] = edits[j], edits[i]
	}
}
