package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
)

type iterator_A struct {
        gc        *githubv4.Client
        since     time.Time
        ctx       context.Context
        err       error
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
        query issueEditQuery_A
}

type timelineIter struct {
        iterVars
        query           timelineQuery
        commentEditIter []commentEditIter
}


type commentEditIter struct {
        iterVars
        //query commentEditQuery
}

type iterVars struct {
        index     int
        capacity  int
        variables varmap
}

type varmap map[string]interface{}


func NewIterator_A(ctx context.Context, client *githubv4.Client, capacity int, owner, project string, since time.Time) *iterator_A {
        i := &iterator_A{
                gc:    client,
                since: since,
                ctx:   ctx,
                issueIter: issueIter{
                        iterVars:      newIterVars(capacity),
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
	}
	i.resetIssueVars()
	return i
}

func newIterVars(capacity int) iterVars {
        return iterVars{
                index:     -1,
                capacity:  capacity,
                variables: varmap{},
        }
}

func (v *varmap) setOwnerProject(owner, project string) {
        (*v)["owner"] = githubv4.String(owner)
        (*v)["name"] = githubv4.String(project)
}

func (i *iterator_A) resetIssueVars() {
        vars := &i.issueIter.variables
        (*vars)["issueFirst"] = githubv4.Int(i.issueIter.capacity)
        (*vars)["issueAfter"] = (*githubv4.String)(nil)
        // I am not sure if the since variable should be used.
        //(*vars)["issueSince"] = githubv4.DateTime{Time: i.since}
        i.issueIter.query.Repository.Issues.PageInfo.HasNextPage = true
        i.issueIter.query.Repository.Issues.PageInfo.EndCursor = ""
}

func (i *iterator_A) resetIssueEditVars() {
        for idx := range i.issueIter.issueEditIter {
                ie := &i.issueIter.issueEditIter[idx]
                ie.variables["issueEditLast"] = githubv4.Int(ie.capacity)
                ie.variables["issueEditBefore"] = (*githubv4.String)(nil)
                ie.query.Node.Issue.UserContentEdits.PageInfo.HasNextPage = true
                ie.query.Node.Issue.UserContentEdits.PageInfo.EndCursor = ""
        }
}

func (i *iterator_A) resetTimelineVars() {
        for idx := range i.issueIter.timelineIter {
                ip := &i.issueIter.timelineIter[idx]
                ip.variables["timelineFirst"] = githubv4.Int(ip.capacity)
                ip.variables["timelineAfter"] = (*githubv4.String)(nil)
                ip.query.Node.Issue.TimelineItems.PageInfo.HasNextPage = true
                ip.query.Node.Issue.TimelineItems.PageInfo.EndCursor = ""
        }
}

func (i *iterator_A) currIssueItem() *issue {
        return &i.issueIter.query.Repository.Issues.Nodes[i.issueIter.index]
}

func (i *iterator_A) currIssueEditIter() *issueEditIter {
        return &i.issueIter.issueEditIter[i.issueIter.index]
}

func (i *iterator_A) currTimelineIter() *timelineIter {
        return &i.issueIter.timelineIter[i.issueIter.index]
}

func (i *iterator_A) currIssueGqlNodeId() githubv4.ID {
        return i.currIssueItem().Id
}

// Error return last encountered error
func (i *iterator_A) Error() error {
        if i.err != nil {
                return i.err
        }
        return i.ctx.Err() // might return nil
}

func (i *iterator_A) HasError() bool {
        return i.err != nil || i.ctx.Err() != nil
}

func (i *iterator_A) NextIssue() bool {
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

func (i *iterator_A) IssueValue() issue {
        return *i.currIssueItem()
}

func (i *iterator_A) queryIssue() bool {
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

func (i *iterator_A) NextIssueEdit() bool {
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

func (i *iterator_A) nextValidIssueEdit() bool {
        // issueEdit.Diff == nil happen if the event is older than early 2018, Github doesn't have the data before that.
        // Best we can do is to ignore the event.
        if issueEdit := i.IssueEditValue(); issueEdit.Diff == nil || string(*issueEdit.Diff) == "" {
                return i.NextIssueEdit()
        }
        return true
}

func (i *iterator_A) IssueEditValue() userContentEdit {
        iei := i.currIssueEditIter()
        return iei.query.Node.Issue.UserContentEdits.Nodes[iei.index]
}

func (i *iterator_A) queryIssueEdit() bool {
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

func (i *iterator_A) NextTimelineItem() bool {
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

func (i *iterator_A) TimelineItemValue() timelineItem {
        tli := i.currTimelineIter()
        return tli.query.Node.Issue.TimelineItems.Nodes[tli.index]
}

func (i *iterator_A) queryTimeline() bool {
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
        //i.resetCommentEditVars()
        timelineItems := &tli.query.Node.Issue.TimelineItems
        if len(timelineItems.Nodes) <= 0 {
                tli.index = -1
                return false
        }
        tli.index = 0
        return true
}



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

	// Alex: It would be really help clearity to get rid of this variable.
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

	// shared context used for all graphql queries
	ctx context.Context

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
func NewIterator(ctx context.Context, client *githubv4.Client, capacity int, owner, project string, since time.Time) *iterator {
	i := &iterator{
		gc:       client,
		since:    since,
		capacity: capacity,
		ctx:      ctx,
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
	i.timeline.variables["issueFirst"] = githubv4.Int(1) // each query one single issue only
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
	for index, ce := range node.TimelineItems.Edges {
		if ce.Node.Typename == "IssueComment" && len(node.TimelineItems.Edges) != 0 {
			reverseEdits(node.TimelineItems.Edges[index].Node.IssueComment.UserContentEdits.Nodes)
		}
	}
}

// Error return last encountered error
func (i *iterator) Error() error {
	return i.err
}

func (i *iterator) queryIssue() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()

	if err := i.gc.Query(ctx, &i.timeline.query, i.timeline.variables); err != nil {
		i.err = err
		return false
	}

	issues := i.timeline.query.Repository.Issues.Nodes
	if len(issues) == 0 {
		return false
	}

	i.reverseTimelineEditNodes()
	return true
}

// NextIssue try to query the next issue and return true. Only one issue is
// queried at each call.
func (i *iterator) NextIssue() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	// if $issueAfter variable is nil we can directly make the first query
	if i.timeline.variables["issueAfter"] == (*githubv4.String)(nil) {
		nextIssue := i.queryIssue()
		// prevent from infinite loop by setting a non nil cursor
		issues := i.timeline.query.Repository.Issues
		i.timeline.variables["issueAfter"] = issues.PageInfo.EndCursor
		return nextIssue
	}

	issues := i.timeline.query.Repository.Issues
	if !issues.PageInfo.HasNextPage {
		return false
	}

	// if we have more issues, query them
	i.timeline.variables["timelineAfter"] = (*githubv4.String)(nil)
	i.timeline.index = -1

	timelineEndCursor := issues.Nodes[0].TimelineItems.PageInfo.EndCursor
	// store cursor for future use
	i.timeline.lastEndCursor = timelineEndCursor

	// query issue block
	nextIssue := i.queryIssue()
	i.timeline.variables["issueAfter"] = issues.PageInfo.EndCursor

	return nextIssue
}

// IssueValue return the actual issue value
func (i *iterator) IssueValue() issueTimeline {
	issues := i.timeline.query.Repository.Issues
	return issues.Nodes[0]
}

// NextTimelineItem return true if there is a next timeline item and increments the index by one.
// It is used iterates over all the timeline items. Extra queries are made if it is necessary.
func (i *iterator) NextTimelineItem() bool {
	if i.err != nil {
		return false
	}

	if i.ctx.Err() != nil {
		return false
	}

	timelineItems := i.timeline.query.Repository.Issues.Nodes[0].TimelineItems
	// after NextIssue call it's good to check wether we have some timelineItems items or not
	// Alex: Correct?
	if len(timelineItems.Edges) == 0 {
		return false
	}

	if i.timeline.index < len(timelineItems.Edges)-1 {
		i.timeline.index++
		return true
	}

	if !timelineItems.PageInfo.HasNextPage {
		return false
	}

	i.timeline.lastEndCursor = timelineItems.PageInfo.EndCursor

	// more timelines, query them
	i.timeline.variables["timelineAfter"] = timelineItems.PageInfo.EndCursor
	// HACK
	var query timelineItemsQuery
	// var variables map[string]interface{}
	variables := make(map[string]interface{})
	variables["owner"] = i.timeline.variables["owner"]
	variables["name"] = i.timeline.variables["name"]
	variables["issueNumber"] = i.timeline.query.Repository.Issues.Nodes[0].Number
	fmt.Println("### Alex using issue number ", i.timeline.query.Repository.Issues.Nodes[0].Number)
	variables["timelineFirst"] = i.timeline.variables["timelineFirst"]
	variables["timelineAfter"] = i.timeline.variables["timelineAfter"]
	variables["commentEditLast"] = i.timeline.variables["commentEditLast"]
	variables["commentEditBefore"] = i.timeline.variables["commentEditBefore"]

	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()

	// if err := i.gc.Query(ctx, &i.timeline.query, i.timeline.variables); err != nil {
	if err := i.gc.Query(ctx, &query, variables); err != nil {
		i.err = err
		return false
	}
	// HACK
	fmt.Println("### Alex after the query")
	i.timeline.variables["timelineFirst"] = variables["timelineFirst"]
	i.timeline.variables["timelineAfter"] = variables["timelineAfter"]
	i.timeline.variables["commentEditLast"] = variables["commentEditLast"]
	i.timeline.variables["commentEditBefore"] = variables["commentEditBefore"]
	i.timeline.query.Repository.Issues.Nodes[0].TimelineItems = query.Repository.Issue.TimelineItems

	timelineItems = i.timeline.query.Repository.Issues.Nodes[0].TimelineItems
	// (in case github returns something weird) just for safety: better return a false than a panic
	if len(timelineItems.Edges) == 0 {
		return false
	}

	i.reverseTimelineEditNodes()
	i.timeline.index = 0
	return true
}

// TimelineItemValue return the actual timeline item value
func (i *iterator) TimelineItemValue() timelineItem {
	timelineItems := i.timeline.query.Repository.Issues.Nodes[0].TimelineItems
	return timelineItems.Edges[i.timeline.index].Node
}

func (i *iterator) queryIssueEdit() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()

	if err := i.gc.Query(ctx, &i.issueEdit.query, i.issueEdit.variables); err != nil {
		i.err = err
		//i.timeline.issueEdit.index = -1
		return false
	}

	issueEdits := i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits
	// reverse issue edits because github
	reverseEdits(issueEdits.Nodes)

	// this is not supposed to happen
	if len(issueEdits.Nodes) == 0 {
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

	if i.ctx.Err() != nil {
		return false
	}

	// this mean we looped over all available issue edits in the timeline.
	// now we have to use i.issueEditQuery
	if i.timeline.issueEdit.index == -2 {
		issueEdits := i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits
		if i.issueEdit.index < len(issueEdits.Nodes)-1 {
			i.issueEdit.index++
			return i.nextValidIssueEdit()
		}

		if !issueEdits.PageInfo.HasPreviousPage {
			i.timeline.issueEdit.index = -1
			i.issueEdit.index = -1
			return false
		}

		// if there is more edits, query them
		i.issueEdit.variables["issueEditBefore"] = issueEdits.PageInfo.StartCursor
		return i.queryIssueEdit()
	}

	issueEdits := i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits
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
	if len(issueEdits.Nodes) == 0 {
		return false
	}

	// loop over them timeline comment edits
	if i.timeline.issueEdit.index < len(issueEdits.Nodes)-1 {
		i.timeline.issueEdit.index++
		return i.nextValidIssueEdit()
	}

	if !issueEdits.PageInfo.HasPreviousPage {
		i.timeline.issueEdit.index = -1
		return false
	}

	// if there is more edits, query them
	i.initIssueEditQueryVariables()
	i.issueEdit.variables["issueEditBefore"] = issueEdits.PageInfo.StartCursor
	return i.queryIssueEdit()
}

// IssueEditValue return the actual issue edit value
func (i *iterator) IssueEditValue() userContentEdit {
	// if we are using issue edit query
	if i.timeline.issueEdit.index == -2 {
		issueEdits := i.issueEdit.query.Repository.Issues.Nodes[0].UserContentEdits
		return issueEdits.Nodes[i.issueEdit.index]
	}

	issueEdits := i.timeline.query.Repository.Issues.Nodes[0].UserContentEdits
	// else get it from timeline issue edit query
	return issueEdits.Nodes[i.timeline.issueEdit.index]
}

func (i *iterator) queryCommentEdit() bool {
	ctx, cancel := context.WithTimeout(i.ctx, defaultTimeout)
	defer cancel()

	if err := i.gc.Query(ctx, &i.commentEdit.query, i.commentEdit.variables); err != nil {
		i.err = err
		return false
	}

	commentEdits := i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits
	// this is not supposed to happen
	if len(commentEdits.Nodes) == 0 {
		i.timeline.commentEdit.index = -1
		return false
	}

	reverseEdits(commentEdits.Nodes)

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

	if i.ctx.Err() != nil {
		return false
	}

	// same as NextIssueEdit
	if i.timeline.commentEdit.index == -2 {
		commentEdits := i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits
		if i.commentEdit.index < len(commentEdits.Nodes)-1 {
			i.commentEdit.index++
			return i.nextValidCommentEdit()
		}

		if !commentEdits.PageInfo.HasPreviousPage {
			i.timeline.commentEdit.index = -1
			i.commentEdit.index = -1
			return false
		}

		// if there is more comment edits, query them
		i.commentEdit.variables["commentEditBefore"] = commentEdits.PageInfo.StartCursor
		return i.queryCommentEdit()
	}

	commentEdits := i.timeline.query.Repository.Issues.Nodes[0].TimelineItems.Edges[i.timeline.index].Node.IssueComment
	// if there is no comment edits
	if len(commentEdits.UserContentEdits.Nodes) == 0 {
		return false
	}

	// loop over them timeline comment edits
	if i.timeline.commentEdit.index < len(commentEdits.UserContentEdits.Nodes)-1 {
		i.timeline.commentEdit.index++
		return i.nextValidCommentEdit()
	}

	if !commentEdits.UserContentEdits.PageInfo.HasPreviousPage {
		i.timeline.commentEdit.index = -1
		return false
	}

	i.initCommentEditQueryVariables()
	if i.timeline.index == 0 {
		i.commentEdit.variables["timelineAfter"] = i.timeline.lastEndCursor
	} else {
		i.commentEdit.variables["timelineAfter"] = i.timeline.query.Repository.Issues.Nodes[0].TimelineItems.Edges[i.timeline.index-1].Cursor
	}

	i.commentEdit.variables["commentEditBefore"] = commentEdits.UserContentEdits.PageInfo.StartCursor

	return i.queryCommentEdit()
}

// CommentEditValue return the actual comment edit value
func (i *iterator) CommentEditValue() userContentEdit {
	if i.timeline.commentEdit.index == -2 {
		return i.commentEdit.query.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits.Nodes[i.commentEdit.index]
	}

	return i.timeline.query.Repository.Issues.Nodes[0].TimelineItems.Edges[i.timeline.index].Node.IssueComment.UserContentEdits.Nodes[i.timeline.commentEdit.index]
}

func reverseEdits(edits []userContentEdit) {
	for i, j := 0, len(edits)-1; i < j; i, j = i+1, j-1 {
		edits[i], edits[j] = edits[j], edits[i]
	}
}
