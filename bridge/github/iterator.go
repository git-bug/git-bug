package github

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
)

type iterator struct {
	// Github graphql client
	gc *githubv4.Client

	// The iterator will only query issues updated or created after the date given in
	// the variable since.
	since time.Time

	// Shared context, which is used for all graphql queries.
	ctx context.Context

	// Sticky error
	err error

	// Issue iterator
	issueIter issueIter
}

type issueIter struct {
	iterVars
	query         issueQuery
	issueEditIter []issueEditIter
	timelineIter  []timelineIter
}

type issueEditIter struct {
	iterVars
	query issueEditQuery
}

type timelineIter struct {
	iterVars
	query           timelineQuery
	commentEditIter []commentEditIter
}

type commentEditIter struct {
	iterVars
	query commentEditQuery
}

type iterVars struct {
	// Iterator index
	index int

	// capacity is the number of elements (issues, issue edits, timeline items, or
	// comment edits) to query at a time. More capacity = more used memory =
	// less queries to make.
	capacity int

	// Variable assignments for graphql query
	variables varmap
}

type varmap map[string]interface{}

func newIterVars(capacity int) iterVars {
	return iterVars{
		index:     -1,
		capacity:  capacity,
		variables: varmap{},
	}
}

// NewIterator creates and initialize a new iterator.
func NewIterator(ctx context.Context, client *githubv4.Client, capacity int, owner, project string, since time.Time) *iterator {
	i := &iterator{
		gc:    client,
		since: since,
		ctx:   ctx,
		issueIter: issueIter{
			iterVars:      newIterVars(capacity),
			timelineIter:  make([]timelineIter, capacity),
			issueEditIter: make([]issueEditIter, capacity),
		},
	}
	i.issueIter.variables.setOwnerProject(owner, project)
	for idx := range i.issueIter.issueEditIter {
		ie := &i.issueIter.issueEditIter[idx]
		ie.iterVars = newIterVars(capacity)
	}
	for i1 := range i.issueIter.timelineIter {
		tli := &i.issueIter.timelineIter[i1]
		tli.iterVars = newIterVars(capacity)
		tli.commentEditIter = make([]commentEditIter, capacity)
		for i2 := range tli.commentEditIter {
			cei := &tli.commentEditIter[i2]
			cei.iterVars = newIterVars(capacity)
		}
	}
	i.resetIssueVars()
	return i
}

func (v *varmap) setOwnerProject(owner, project string) {
	(*v)["owner"] = githubv4.String(owner)
	(*v)["name"] = githubv4.String(project)
}

func (i *iterator) resetIssueVars() {
	vars := &i.issueIter.variables
	(*vars)["issueFirst"] = githubv4.Int(i.issueIter.capacity)
	(*vars)["issueAfter"] = (*githubv4.String)(nil)
	(*vars)["issueSince"] = githubv4.DateTime{Time: i.since}
	i.issueIter.query.Repository.Issues.PageInfo.HasNextPage = true
	i.issueIter.query.Repository.Issues.PageInfo.EndCursor = ""
}

func (i *iterator) resetIssueEditVars() {
	for idx := range i.issueIter.issueEditIter {
		ie := &i.issueIter.issueEditIter[idx]
		ie.variables["issueEditLast"] = githubv4.Int(ie.capacity)
		ie.variables["issueEditBefore"] = (*githubv4.String)(nil)
		ie.query.Node.Issue.UserContentEdits.PageInfo.HasNextPage = true
		ie.query.Node.Issue.UserContentEdits.PageInfo.EndCursor = ""
	}
}

func (i *iterator) resetTimelineVars() {
	for idx := range i.issueIter.timelineIter {
		ip := &i.issueIter.timelineIter[idx]
		ip.variables["timelineFirst"] = githubv4.Int(ip.capacity)
		ip.variables["timelineAfter"] = (*githubv4.String)(nil)
		ip.query.Node.Issue.TimelineItems.PageInfo.HasNextPage = true
		ip.query.Node.Issue.TimelineItems.PageInfo.EndCursor = ""
	}
}

func (i *iterator) resetCommentEditVars() {
	for i1 := range i.issueIter.timelineIter {
		for i2 := range i.issueIter.timelineIter[i1].commentEditIter {
			ce := &i.issueIter.timelineIter[i1].commentEditIter[i2]
			ce.variables["commentEditLast"] = githubv4.Int(ce.capacity)
			ce.variables["commentEditBefore"] = (*githubv4.String)(nil)
			ce.query.Node.IssueComment.UserContentEdits.PageInfo.HasNextPage = true
			ce.query.Node.IssueComment.UserContentEdits.PageInfo.EndCursor = ""
		}
	}
}

// Error return last encountered error
func (i *iterator) Error() error {
	if i.err != nil {
		return i.err
	}
	return i.ctx.Err() // might return nil
}

func (i *iterator) HasError() bool {
	return i.err != nil || i.ctx.Err() != nil
}

func (i *iterator) currIssueItem() *issue {
	return &i.issueIter.query.Repository.Issues.Nodes[i.issueIter.index]
}

func (i *iterator) currIssueEditIter() *issueEditIter {
	return &i.issueIter.issueEditIter[i.issueIter.index]
}

func (i *iterator) currTimelineIter() *timelineIter {
	return &i.issueIter.timelineIter[i.issueIter.index]
}

func (i *iterator) currCommentEditIter() *commentEditIter {
	timelineIter := i.currTimelineIter()
	return &timelineIter.commentEditIter[timelineIter.index]
}

func (i *iterator) currIssueGqlNodeId() githubv4.ID {
	return i.currIssueItem().Id
}

// NextIssue returns true if there exists a next issue and advances the iterator by one.
// It is used to iterate over all issues. Queries to github are made when necessary.
func (i *iterator) NextIssue() bool {
	if i.HasError() {
		return false
	}
	index := &i.issueIter.index
	issues := &i.issueIter.query.Repository.Issues
	issueItems := &issues.Nodes
	if 0 <= *index && *index < len(*issueItems)-1 {
		*index += 1
		return true
	}

	if !issues.PageInfo.HasNextPage {
		return false
	}
	nextIssue := i.queryIssue()
	return nextIssue
}

// IssueValue returns the actual issue value.
func (i *iterator) IssueValue() issue {
	return *i.currIssueItem()
}

func (i *iterator) queryIssue() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()
	if endCursor := i.issueIter.query.Repository.Issues.PageInfo.EndCursor; endCursor != "" {
		i.issueIter.variables["issueAfter"] = endCursor
	}
	if err := i.gc.Query(ctx, &i.issueIter.query, i.issueIter.variables); err != nil {
		i.err = err
		return false
	}
	i.resetIssueEditVars()
	i.resetTimelineVars()
	issueItems := &i.issueIter.query.Repository.Issues.Nodes
	if len(*issueItems) <= 0 {
		i.issueIter.index = -1
		return false
	}
	i.issueIter.index = 0
	return true
}

// NextIssueEdit returns true if there exists a next issue edit and advances the iterator
// by one. It is used to iterate over all the issue edits. Queries to github are made when
// necessary.
func (i *iterator) NextIssueEdit() bool {
	if i.HasError() {
		return false
	}
	ieIter := i.currIssueEditIter()
	ieIdx := &ieIter.index
	ieItems := ieIter.query.Node.Issue.UserContentEdits
	if 0 <= *ieIdx && *ieIdx < len(ieItems.Nodes)-1 {
		*ieIdx += 1
		return i.nextValidIssueEdit()
	}
	if !ieItems.PageInfo.HasNextPage {
		return false
	}
	querySucc := i.queryIssueEdit()
	if !querySucc {
		return false
	}
	return i.nextValidIssueEdit()
}

func (i *iterator) nextValidIssueEdit() bool {
	// issueEdit.Diff == nil happen if the event is older than early 2018, Github doesn't have
	// the data before that. Best we can do is to ignore the event.
	if issueEdit := i.IssueEditValue(); issueEdit.Diff == nil || string(*issueEdit.Diff) == "" {
		return i.NextIssueEdit()
	}
	return true
}

// IssueEditValue returns the actual issue edit value.
func (i *iterator) IssueEditValue() userContentEdit {
	iei := i.currIssueEditIter()
	return iei.query.Node.Issue.UserContentEdits.Nodes[iei.index]
}

func (i *iterator) queryIssueEdit() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()
	iei := i.currIssueEditIter()
	if endCursor := iei.query.Node.Issue.UserContentEdits.PageInfo.EndCursor; endCursor != "" {
		iei.variables["issueEditBefore"] = endCursor
	}
	iei.variables["gqlNodeId"] = i.currIssueGqlNodeId()
	if err := i.gc.Query(ctx, &iei.query, iei.variables); err != nil {
		i.err = err
		return false
	}
	issueEditItems := iei.query.Node.Issue.UserContentEdits.Nodes
	if len(issueEditItems) <= 0 {
		iei.index = -1
		return false
	}
	// The UserContentEditConnection in the Github API serves its elements in reverse chronological
	// order. For our purpose we have to reverse the edits.
	reverseEdits(issueEditItems)
	iei.index = 0
	return true
}

// NextTimelineItem returns true if there exists a next timeline item and advances the iterator
// by one. It is used to iterate over all the timeline items. Queries to github are made when
// necessary.
func (i *iterator) NextTimelineItem() bool {
	if i.HasError() {
		return false
	}
	tlIter := &i.issueIter.timelineIter[i.issueIter.index]
	tlIdx := &tlIter.index
	tlItems := tlIter.query.Node.Issue.TimelineItems
	if 0 <= *tlIdx && *tlIdx < len(tlItems.Nodes)-1 {
		*tlIdx += 1
		return true
	}
	if !tlItems.PageInfo.HasNextPage {
		return false
	}
	nextTlItem := i.queryTimeline()
	return nextTlItem
}

// TimelineItemValue returns the actual timeline item value.
func (i *iterator) TimelineItemValue() timelineItem {
	tli := i.currTimelineIter()
	return tli.query.Node.Issue.TimelineItems.Nodes[tli.index]
}

func (i *iterator) queryTimeline() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()
	tli := i.currTimelineIter()
	if endCursor := tli.query.Node.Issue.TimelineItems.PageInfo.EndCursor; endCursor != "" {
		tli.variables["timelineAfter"] = endCursor
	}
	tli.variables["gqlNodeId"] = i.currIssueGqlNodeId()
	if err := i.gc.Query(ctx, &tli.query, tli.variables); err != nil {
		i.err = err
		return false
	}
	i.resetCommentEditVars()
	timelineItems := &tli.query.Node.Issue.TimelineItems
	if len(timelineItems.Nodes) <= 0 {
		tli.index = -1
		return false
	}
	tli.index = 0
	return true
}

// NextCommentEdit returns true if there exists a next comment edit and advances the iterator
// by one. It is used to iterate over all issue edits. Queries to github are made when
// necessary.
func (i *iterator) NextCommentEdit() bool {
	if i.HasError() {
		return false
	}

	tmlnVal := i.TimelineItemValue()
	if tmlnVal.Typename != "IssueComment" {
		// The timeline iterator does not point to a comment.
		i.err = errors.New("Call to NextCommentEdit() while timeline item is not a comment")
		return false
	}

	cei := i.currCommentEditIter()
	ceIdx := &cei.index
	ceItems := &cei.query.Node.IssueComment.UserContentEdits
	if 0 <= *ceIdx && *ceIdx < len(ceItems.Nodes)-1 {
		*ceIdx += 1
		return i.nextValidCommentEdit()
	}
	if !ceItems.PageInfo.HasNextPage {
		return false
	}
	querySucc := i.queryCommentEdit()
	if !querySucc {
		return false
	}
	return i.nextValidCommentEdit()
}

func (i *iterator) nextValidCommentEdit() bool {
	// if comment edit diff is a nil pointer or points to an empty string look for next value
	if commentEdit := i.CommentEditValue(); commentEdit.Diff == nil || string(*commentEdit.Diff) == "" {
		return i.NextCommentEdit()
	}
	return true
}

// CommentEditValue returns the actual comment edit value.
func (i *iterator) CommentEditValue() userContentEdit {
	cei := i.currCommentEditIter()
	return cei.query.Node.IssueComment.UserContentEdits.Nodes[cei.index]
}

func (i *iterator) queryCommentEdit() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()
	cei := i.currCommentEditIter()

	if endCursor := cei.query.Node.IssueComment.UserContentEdits.PageInfo.EndCursor; endCursor != "" {
		cei.variables["commentEditBefore"] = endCursor
	}
	tmlnVal := i.TimelineItemValue()
	if tmlnVal.Typename != "IssueComment" {
		i.err = errors.New("Call to queryCommentEdit() while timeline item is not a comment")
		return false
	}
	cei.variables["gqlNodeId"] = tmlnVal.IssueComment.Id
	if err := i.gc.Query(ctx, &cei.query, cei.variables); err != nil {
		i.err = err
		return false
	}
	ceItems := cei.query.Node.IssueComment.UserContentEdits.Nodes
	if len(ceItems) <= 0 {
		cei.index = -1
		return false
	}
	// The UserContentEditConnection in the Github API serves its elements in reverse chronological
	// order. For our purpose we have to reverse the edits.
	reverseEdits(ceItems)
	cei.index = 0
	return true
}

func reverseEdits(edits []userContentEdit) {
	for i, j := 0, len(edits)-1; i < j; i, j = i+1, j-1 {
		edits[i], edits[j] = edits[j], edits[i]
	}
}
