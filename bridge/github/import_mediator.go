package github

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shurcooL/githubv4"
)

type varmap map[string]interface{}

func trace() {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	fmt.Printf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
}

const (
	NUM_ISSUES         = 50
	NUM_ISSUE_EDITS    = 99
	NUM_TIMELINE_ITEMS = 99
	NUM_COMMENT_EDITS  = 99

	CHAN_CAPACITY = 128
)

// TODO: remove all debug output and trace() in all files. Use ag

type importMediator struct {
	// Github graphql client
	gc      *githubv4.Client
	owner   string
	project string
	// The iterator will only query issues updated or created after the date given in
	// the variable since.
	since time.Time

	issues           chan issue
	issueEditsMut    sync.Mutex
	timelineItemsMut sync.Mutex
	commentEditsMut  sync.Mutex
	issueEdits       map[githubv4.ID]chan userContentEdit
	timelineItems    map[githubv4.ID]chan timelineItem
	commentEdits     map[githubv4.ID]chan userContentEdit

	// Sticky error
	err error
}

func NewImportMediator(ctx context.Context, client *githubv4.Client, owner, project string, since time.Time) *importMediator {
	mm := importMediator{
		gc:               client,
		owner:            owner,
		project:          project,
		since:            since,
		issues:           make(chan issue, CHAN_CAPACITY),
		issueEditsMut:    sync.Mutex{},
		timelineItemsMut: sync.Mutex{},
		commentEditsMut:  sync.Mutex{},
		issueEdits:       make(map[githubv4.ID]chan userContentEdit),
		timelineItems:    make(map[githubv4.ID]chan timelineItem),
		commentEdits:     make(map[githubv4.ID]chan userContentEdit),
		err:              nil,
	}
	go func() {
		defer close(mm.issues)
		mm.fillChannels(ctx)
	}()
	return &mm
}

func (mm *importMediator) Issues() <-chan issue {
	return mm.issues
}

func (mm *importMediator) IssueEdits(issue *issue) <-chan userContentEdit {
	mm.issueEditsMut.Lock()
	channel := mm.issueEdits[issue.Id]
	mm.issueEditsMut.Unlock()
	return channel
}

func (mm *importMediator) TimelineItems(issue *issue) <-chan timelineItem {
	mm.timelineItemsMut.Lock()
	channel := mm.timelineItems[issue.Id]
	mm.timelineItemsMut.Unlock()
	return channel
}

func (mm *importMediator) CommentEdits(comment *issueComment) <-chan userContentEdit {
	mm.commentEditsMut.Lock()
	channel := mm.commentEdits[comment.Id]
	mm.commentEditsMut.Unlock()
	return channel
}

func (mm *importMediator) Error() error {
	return mm.err
}

func (mm *importMediator) User(ctx context.Context, loginName string) (*user, error) {
	query := userQuery{}
	vars := varmap{"login": githubv4.String(loginName)}
	c, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	if err := mm.mQuery(c, &query, vars); err != nil {
		return nil, err
	}
	return &query.User, nil
}

func (mm *importMediator) fillChannels(ctx context.Context) {
	issueCursor := githubv4.String("")
	for {
		issues, hasIssues := mm.queryIssue(ctx, issueCursor)
		if !hasIssues {
			break
		}
		issueCursor = issues.PageInfo.EndCursor
		for _, issueNode := range issues.Nodes {
			// fmt.Printf(">>> issue: %v\n", issueNode.issue.Title)
			mm.fillChannelIssueEdits(ctx, &issueNode)
			mm.fillChannelTimeline(ctx, &issueNode)
			// To avoid race conditions add the issue only after all its edits,
			// timeline times, etc. are added to their respective channels.
			mm.issues <- issueNode.issue
		}
	}
}

func (mm *importMediator) fillChannelIssueEdits(ctx context.Context, issueNode *issueNode) {
	// fmt.Printf("fillChannelIssueEdit() issue id == %v\n", issueNode.issue.Id)
	// fmt.Printf("%v\n", issueNode)
	channel := make(chan userContentEdit, CHAN_CAPACITY)
	defer close(channel)
	mm.issueEditsMut.Lock()
	mm.issueEdits[issueNode.issue.Id] = channel
	mm.issueEditsMut.Unlock()
	edits := &issueNode.UserContentEdits
	hasEdits := true
	for hasEdits {
		// fmt.Println("before the reversed loop")
		for edit := range reverse(edits.Nodes) {
			// fmt.Println("in the reversed loop")
			if edit.Diff == nil || string(*edit.Diff) == "" {
				// issueEdit.Diff == nil happen if the event is older than
				// early 2018, Github doesn't have the data before that.
				// Best we can do is to ignore the event.
				continue
			}
			// fmt.Printf("about to push issue edit\n")
			channel <- edit
		}
		// fmt.Printf("has next ? %v\n", edits.PageInfo.HasNextPage)
		// fmt.Printf("has previous ? %v\n", edits.PageInfo.HasPreviousPage)
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, hasEdits = mm.queryIssueEdits(ctx, issueNode.issue.Id, edits.PageInfo.EndCursor)
	}
}

func (mm *importMediator) fillChannelTimeline(ctx context.Context, issueNode *issueNode) {
	// fmt.Printf("fullChannelTimeline()\n")
	channel := make(chan timelineItem, CHAN_CAPACITY)
	defer close(channel)
	mm.timelineItemsMut.Lock()
	mm.timelineItems[issueNode.issue.Id] = channel
	mm.timelineItemsMut.Unlock()
	items := &issueNode.TimelineItems
	hasItems := true
	for hasItems {
		for _, item := range items.Nodes {
			channel <- item
			mm.fillChannelCommentEdits(ctx, &item)
		}
		// fmt.Printf("has next ? %v\n", items.PageInfo.HasNextPage)
		// fmt.Printf("has previous ? %v\n", items.PageInfo.HasPreviousPage)
		if !items.PageInfo.HasNextPage {
			break
		}
		items, hasItems = mm.queryTimelineItems(ctx, issueNode.issue.Id, items.PageInfo.EndCursor)
	}
}

func (mm *importMediator) fillChannelCommentEdits(ctx context.Context, item *timelineItem) {
	// This concerns only timeline items of type comment
	if item.Typename != "IssueComment" {
		return
	}
	comment := &item.IssueComment
	channel := make(chan userContentEdit, CHAN_CAPACITY)
	defer close(channel)
	mm.commentEditsMut.Lock()
	mm.commentEdits[comment.Id] = channel
	mm.commentEditsMut.Unlock()
	edits := &comment.UserContentEdits
	hasEdits := true
	for hasEdits {
		for edit := range reverse(edits.Nodes) {
			if edit.Diff == nil || string(*edit.Diff) == "" {
				// issueEdit.Diff == nil happen if the event is older than
				// early 2018, Github doesn't have the data before that.
				// Best we can do is to ignore the event.
				continue
			}
			channel <- edit
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, hasEdits = mm.queryCommentEdits(ctx, comment.Id, edits.PageInfo.EndCursor)
	}
}

func (mm *importMediator) queryCommentEdits(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*userContentEditConnection, bool) {
	// trace()
	vars := varmap{
		"gqlNodeId":       nid,
		"commentEditLast": githubv4.Int(NUM_COMMENT_EDITS),
	}
	if cursor == "" {
		vars["commentEditBefore"] = (*githubv4.String)(nil)
	} else {
		vars["commentEditBefore"] = cursor
	}
	c, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	query := commentEditQuery{}
	if err := mm.mQuery(c, &query, vars); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.IssueComment.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) queryTimelineItems(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*timelineItemsConnection, bool) {
	// trace()
	vars := varmap{
		"gqlNodeId":         nid,
		"timelineFirst":     githubv4.Int(NUM_TIMELINE_ITEMS),
		"commentEditLast":   githubv4.Int(NUM_COMMENT_EDITS),
		"commentEditBefore": (*githubv4.String)(nil),
	}
	if cursor == "" {
		vars["timelineAfter"] = (*githubv4.String)(nil)
	} else {
		vars["timelineAfter"] = cursor
	}
	c, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	query := timelineQuery{}
	if err := mm.mQuery(c, &query, vars); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.Issue.TimelineItems
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) queryIssueEdits(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*userContentEditConnection, bool) {
	// trace()
	vars := varmap{
		"gqlNodeId":     nid,
		"issueEditLast": githubv4.Int(NUM_ISSUE_EDITS),
	}
	if cursor == "" {
		vars["issueEditBefore"] = (*githubv4.String)(nil)
	} else {
		vars["issueEditBefore"] = cursor
	}
	c, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	query := issueEditQuery{}
	if err := mm.mQuery(c, &query, vars); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.Issue.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) queryIssue(ctx context.Context, cursor githubv4.String) (*issueConnection, bool) {
	// trace()
	vars := varmap{
		"owner":             githubv4.String(mm.owner),
		"name":              githubv4.String(mm.project),
		"issueSince":        githubv4.DateTime{Time: mm.since},
		"issueFirst":        githubv4.Int(NUM_ISSUES),
		"issueEditLast":     githubv4.Int(NUM_ISSUE_EDITS),
		"issueEditBefore":   (*githubv4.String)(nil),
		"timelineFirst":     githubv4.Int(NUM_TIMELINE_ITEMS),
		"timelineAfter":     (*githubv4.String)(nil),
		"commentEditLast":   githubv4.Int(NUM_COMMENT_EDITS),
		"commentEditBefore": (*githubv4.String)(nil),
	}
	if cursor == "" {
		vars["issueAfter"] = (*githubv4.String)(nil)
	} else {
		vars["issueAfter"] = githubv4.String(cursor)
	}
	c, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	query := issueQuery{}
	if err := mm.mQuery(c, &query, vars); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Repository.Issues
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func reverse(eds []userContentEdit) chan userContentEdit {
	ret := make(chan userContentEdit)
	go func() {
		for i := range eds {
			ret <- eds[len(eds)-1-i]
		}
		close(ret)
	}()
	return ret
}

type rateLimiter interface {
	rateLimit() rateLimit
}

// TODO: move that into its own file
//
// mQuery executes a single GraphQL query. The variable query is used to derive the GraphQL
// query and it is used to populate the response into it. It should be a pointer to a struct
// that corresponds to the Github graphql schema and it should implement the rateLimiter
// interface. This function queries Github for the remaining rate limit points before
// executing the actual query. The function waits, if there are not enough rate limiting
// points left.
func (mm *importMediator) mQuery(ctx context.Context, query rateLimiter, vars map[string]interface{}) error {
	// First: check the cost of the query and wait if necessary
	vars["dryRun"] = githubv4.Boolean(true)
	qctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	if err := mm.gc.Query(qctx, query, vars); err != nil {
		return err
	}
	fmt.Printf("%v\n", query)
	rateLimit := query.rateLimit()
	if rateLimit.Cost > rateLimit.Remaining {
		resetTime := rateLimit.ResetAt.Time
		fmt.Println("Github rate limit exhausted")
		fmt.Printf("Sleeping until %s\n", resetTime.String())
		// Add a few seconds (8) for good measure
		timer := time.NewTimer(time.Until(resetTime.Add(8 * time.Second)))
		select {
		case <-ctx.Done():
			stop(timer)
			return ctx.Err()
		case <-timer.C:
		}
	}
	// Second: Do the actual query
	vars["dryRun"] = githubv4.Boolean(false)
	qctx, cancel = context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	if err := mm.gc.Query(qctx, query, vars); err != nil {
		return err
	}
	return nil
}

func stop(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
