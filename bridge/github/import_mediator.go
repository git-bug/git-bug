package github

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shurcooL/githubv4"
)

const ( // These values influence how fast the github graphql rate limit is exhausted.
	NUM_ISSUES         = 40
	NUM_ISSUE_EDITS    = 100
	NUM_TIMELINE_ITEMS = 100
	NUM_COMMENT_EDITS  = 100

	CHAN_CAPACITY = 128
)

// importMediator provides a convenient interface to retrieve issues from the Github GraphQL API.
type importMediator struct {
	// Github graphql client
	gc *githubv4.Client

	// name of the repository owner on Github
	owner string

	// name of the Github repository
	project string

	// since specifies which issues to import. Issues that have been updated at or after the
	// given date should be imported.
	since time.Time

	// issues is a channel holding bundles of issues to be imported. Each bundle holds the data
	// associated with one issue.
	issues chan issueBundle

	// Sticky error
	err error

	// errMut is a mutex to synchronize access to the sticky error variable err.
	errMut sync.Mutex
}

type issueBundle struct {
	issue           issue
	issueEdits      <-chan userContentEdit
	timelineBundles <-chan timelineBundle
}

type timelineBundle struct {
	timelineItem     timelineItem
	userContentEdits <-chan userContentEdit
}

func (mm *importMediator) setError(err error) {
	mm.errMut.Lock()
	mm.err = err
	mm.errMut.Unlock()
}

func NewImportMediator(ctx context.Context, client *githubv4.Client, owner, project string, since time.Time) *importMediator {
	mm := importMediator{
		gc:      client,
		owner:   owner,
		project: project,
		since:   since,
		issues:  make(chan issueBundle, CHAN_CAPACITY),
		err:     nil,
	}
	go func() {
		mm.fillIssues(ctx)
		close(mm.issues)
	}()
	return &mm
}

type varmap map[string]interface{}

func newIssueVars(owner, project string, since time.Time) varmap {
	return varmap{
		"owner":             githubv4.String(owner),
		"name":              githubv4.String(project),
		"issueSince":        githubv4.DateTime{Time: since},
		"issueFirst":        githubv4.Int(NUM_ISSUES),
		"issueEditLast":     githubv4.Int(NUM_ISSUE_EDITS),
		"issueEditBefore":   (*githubv4.String)(nil),
		"timelineFirst":     githubv4.Int(NUM_TIMELINE_ITEMS),
		"timelineAfter":     (*githubv4.String)(nil),
		"commentEditLast":   githubv4.Int(NUM_COMMENT_EDITS),
		"commentEditBefore": (*githubv4.String)(nil),
	}
}

func newIssueEditVars() varmap {
	return varmap{
		"issueEditLast": githubv4.Int(NUM_ISSUE_EDITS),
	}
}

func newTimelineVars() varmap {
	return varmap{
		"timelineFirst":     githubv4.Int(NUM_TIMELINE_ITEMS),
		"commentEditLast":   githubv4.Int(NUM_COMMENT_EDITS),
		"commentEditBefore": (*githubv4.String)(nil),
	}
}

func newCommentEditVars() varmap {
	return varmap{
		"commentEditLast": githubv4.Int(NUM_COMMENT_EDITS),
	}
}

func (mm *importMediator) Issues() <-chan issueBundle {
	return mm.issues
}

func (mm *importMediator) Error() error {
	mm.errMut.Lock()
	err := mm.err
	mm.errMut.Unlock()
	return err
}

func (mm *importMediator) User(ctx context.Context, loginName string) (*user, error) {
	query := userQuery{}
	vars := varmap{"login": githubv4.String(loginName)}
	if err := mm.mQuery(ctx, &query, vars); err != nil {
		return nil, err
	}
	return &query.User, nil
}

func (mm *importMediator) fillIssues(ctx context.Context) {
	initialCursor := githubv4.String("")
	issues, hasIssues := mm.queryIssue(ctx, initialCursor)
	for hasIssues {
		for _, node := range issues.Nodes {
			// We need to send an issue-bundle over the issue channel before we start
			// filling the issue edits and the timeline items to avoid deadlocks.
			issueEditChan := make(chan userContentEdit, CHAN_CAPACITY)
			timelineBundleChan := make(chan timelineBundle, CHAN_CAPACITY)
			select {
			case <-ctx.Done():
				return
			case mm.issues <- issueBundle{node.issue, issueEditChan, timelineBundleChan}:
			}

			// We do not know whether the client reads from the issue edit channel
			// or the timeline channel first. Since the capacity of any channel is limited
			// any send operation may block. Hence, in order to avoid deadlocks we need
			// to send over both these channels concurrently.
			go func(node issueNode) {
				mm.fillIssueEdits(ctx, &node, issueEditChan)
				close(issueEditChan)
			}(node)
			go func(node issueNode) {
				mm.fillTimeline(ctx, &node, timelineBundleChan)
				close(timelineBundleChan)
			}(node)
		}
		if !issues.PageInfo.HasNextPage {
			break
		}
		issues, hasIssues = mm.queryIssue(ctx, issues.PageInfo.EndCursor)
	}
}

func (mm *importMediator) fillIssueEdits(ctx context.Context, issueNode *issueNode, channel chan userContentEdit) {
	edits := &issueNode.UserContentEdits
	hasEdits := true
	for hasEdits {
		for edit := range reverse(edits.Nodes) {
			if edit.Diff == nil || string(*edit.Diff) == "" {
				// issueEdit.Diff == nil happen if the event is older than early
				// 2018, Github doesn't have the data before that. Best we can do is
				// to ignore the event.
				continue
			}
			select {
			case <-ctx.Done():
				return
			case channel <- edit:
			}
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, hasEdits = mm.queryIssueEdits(ctx, issueNode.issue.Id, edits.PageInfo.EndCursor)
	}
}

func (mm *importMediator) fillTimeline(ctx context.Context, issueNode *issueNode, channel chan timelineBundle) {
	items := &issueNode.TimelineItems
	hasItems := true
	for hasItems {
		for _, item := range items.Nodes {
			if item.Typename == "IssueComment" {
				// Issue comments are different than other timeline items in that
				// they may have associated user content edits.
				//
				// Send over the timeline-channel before starting to fill the comment
				// edits.
				commentEditChan := make(chan userContentEdit, CHAN_CAPACITY)
				select {
				case <-ctx.Done():
					return
				case channel <- timelineBundle{item, commentEditChan}:
				}
				// We need to create a new goroutine for filling the comment edit
				// channel.
				go func(item timelineItem) {
					mm.fillCommentEdits(ctx, &item, commentEditChan)
					close(commentEditChan)
				}(item)
			} else {
				select {
				case <-ctx.Done():
					return
				case channel <- timelineBundle{item, nil}:
				}
			}
		}
		if !items.PageInfo.HasNextPage {
			break
		}
		items, hasItems = mm.queryTimeline(ctx, issueNode.issue.Id, items.PageInfo.EndCursor)
	}
}

func (mm *importMediator) fillCommentEdits(ctx context.Context, item *timelineItem, channel chan userContentEdit) {
	// Here we are only concerned with timeline items of type issueComment.
	if item.Typename != "IssueComment" {
		return
	}
	comment := &item.IssueComment
	edits := &comment.UserContentEdits
	hasEdits := true
	for hasEdits {
		for edit := range reverse(edits.Nodes) {
			if edit.Diff == nil || string(*edit.Diff) == "" {
				// issueEdit.Diff == nil happen if the event is older than early
				// 2018, Github doesn't have the data before that. Best we can do is
				// to ignore the event.
				continue
			}
			select {
			case <-ctx.Done():
				return
			case channel <- edit:
			}
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, hasEdits = mm.queryCommentEdits(ctx, comment.Id, edits.PageInfo.EndCursor)
	}
}

func (mm *importMediator) queryCommentEdits(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*userContentEditConnection, bool) {
	vars := newCommentEditVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["commentEditBefore"] = (*githubv4.String)(nil)
	} else {
		vars["commentEditBefore"] = cursor
	}
	query := commentEditQuery{}
	if err := mm.mQuery(ctx, &query, vars); err != nil {
		mm.setError(err)
		return nil, false
	}
	connection := &query.Node.IssueComment.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) queryTimeline(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*timelineItemsConnection, bool) {
	vars := newTimelineVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["timelineAfter"] = (*githubv4.String)(nil)
	} else {
		vars["timelineAfter"] = cursor
	}
	query := timelineQuery{}
	if err := mm.mQuery(ctx, &query, vars); err != nil {
		mm.setError(err)
		return nil, false
	}
	connection := &query.Node.Issue.TimelineItems
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) queryIssueEdits(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*userContentEditConnection, bool) {
	vars := newIssueEditVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["issueEditBefore"] = (*githubv4.String)(nil)
	} else {
		vars["issueEditBefore"] = cursor
	}
	query := issueEditQuery{}
	if err := mm.mQuery(ctx, &query, vars); err != nil {
		mm.setError(err)
		return nil, false
	}
	connection := &query.Node.Issue.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) queryIssue(ctx context.Context, cursor githubv4.String) (*issueConnection, bool) {
	vars := newIssueVars(mm.owner, mm.project, mm.since)
	if cursor == "" {
		vars["issueAfter"] = (*githubv4.String)(nil)
	} else {
		vars["issueAfter"] = githubv4.String(cursor)
	}
	query := issueQuery{}
	if err := mm.mQuery(ctx, &query, vars); err != nil {
		mm.setError(err)
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

// mQuery executes a single GraphQL query. The variable query is used to derive the GraphQL query
// and it is used to populate the response into it. It should be a pointer to a struct that
// corresponds to the Github graphql schema and it has to implement the rateLimiter interface. If
// there is a Github rate limiting error, then the function sleeps and retries after the rate limit
// is expired.
func (mm *importMediator) mQuery(ctx context.Context, query rateLimiter, vars map[string]interface{}) error {
	// first: just send the query to the graphql api
	vars["dryRun"] = githubv4.Boolean(false)
	qctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err := mm.gc.Query(qctx, query, vars)
	if err == nil {
		// no error: done
		return nil
	}
	// matching the error string
	if !strings.Contains(err.Error(), "API rate limit exceeded") {
		// an error, but not the API rate limit error: done
		return err
	}
	// a rate limit error
	// ask the graphql api for rate limiting information
	vars["dryRun"] = githubv4.Boolean(true)
	qctx, cancel = context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	if err := mm.gc.Query(qctx, query, vars); err != nil {
		return err
	}
	rateLimit := query.rateLimit()
	if rateLimit.Cost > rateLimit.Remaining {
		// sleep
		resetTime := rateLimit.ResetAt.Time
		// Add a few seconds (8) for good measure
		resetTime = resetTime.Add(8 * time.Second)
		fmt.Printf("Github rate limit exhausted. Sleeping until %s\n", resetTime.String())
		timer := time.NewTimer(time.Until(resetTime))
		select {
		case <-ctx.Done():
			stop(timer)
			return ctx.Err()
		case <-timer.C:
		}
	}
	// run the original query again
	vars["dryRun"] = githubv4.Boolean(false)
	qctx, cancel = context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err = mm.gc.Query(qctx, query, vars)
	return err // might be nil
}

func stop(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
