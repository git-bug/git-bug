package github

import (
	"context"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/shurcooL/githubv4"
)

// Set GITBUG_GITHUB_RATELIMIT_LOG=1 to have each top-level GraphQL query log
// its cost and remaining budget to stderr. Off by default so normal runs are
// quiet; invaluable when diagnosing rate-limit starvation.
var logRateLimit = os.Getenv("GITBUG_GITHUB_RATELIMIT_LOG") == "1"

// Most recent budget observation across all mediators — purely for debug.
var lastRateLimit atomic.Value // stores rateLimit

func noteRateLimit(queryName, owner, project string, rl rateLimit) {
	lastRateLimit.Store(rl)
	if !logRateLimit {
		return
	}
	log.Printf(
		"github.graphql %s/%s %s: cost=%d remaining=%d/%d resetsAt=%s",
		owner, project, queryName, int(rl.Cost), int(rl.Remaining), int(rl.Limit),
		rl.ResetAt.Format(time.RFC3339),
	)
}

// LastRateLimit returns the most recent rateLimit observed across any
// bridge sync (zero value if we've never successfully queried yet).
func LastRateLimit() (cost, remaining, limit int, resetAt time.Time) {
	v, _ := lastRateLimit.Load().(rateLimit)
	return int(v.Cost), int(v.Remaining), int(v.Limit), v.ResetAt.Time
}

const (
	// These values influence how fast the github graphql rate limit is exhausted.

	NumIssues         = 40
	NumIssueEdits     = 100
	NumTimelineItems  = 100
	NumCommentEdits   = 100
	NumReviewComments = 50

	ChanCapacity = 128
)

// importMediator provides a convenient interface to retrieve issues from the Github GraphQL API.
type importMediator struct {
	// Github graphql client
	gh *rateLimitHandlerClient

	// name of the repository owner on Github
	owner string

	// name of the Github repository
	project string

	// since specifies which issues to import. Issues that have been updated at or after the
	// given date should be imported.
	since time.Time

	// importEvents holds events representing issues, comments, edits, ...
	// In this channel issues are immediately followed by their issue edits and comments are
	// immediately followed by their comment edits.
	importEvents chan ImportEvent

	// Sticky error
	err error
}

func NewImportMediator(ctx context.Context, client *rateLimitHandlerClient, owner, project string, since time.Time) *importMediator {
	mm := importMediator{
		gh:           client,
		owner:        owner,
		project:      project,
		since:        since,
		importEvents: make(chan ImportEvent, ChanCapacity),
		err:          nil,
	}

	go mm.start(ctx)

	return &mm
}

func (mm *importMediator) start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	mm.fillImportEvents(ctx)
	// Make sure we cancel everything when we are done, instead of relying on the parent context
	// This should unblock pending send to the channel if the capacity was reached and avoid a panic/race when closing.
	cancel()
	close(mm.importEvents)
}

// NextImportEvent returns the next ImportEvent, or nil if done.
func (mm *importMediator) NextImportEvent() ImportEvent {
	return <-mm.importEvents
}

func (mm *importMediator) Error() error {
	return mm.err
}

func (mm *importMediator) User(ctx context.Context, loginName string) (*user, error) {
	query := userQuery{}
	vars := varmap{"login": githubv4.String(loginName)}
	if err := mm.gh.queryImport(ctx, &query, vars, mm.importEvents); err != nil {
		return nil, err
	}
	return &query.User, nil
}

func (mm *importMediator) fillImportEvents(ctx context.Context) {
	initialCursor := githubv4.String("")
	issues, hasIssues := mm.queryIssue(ctx, initialCursor)
	for hasIssues {
		for _, node := range issues.Nodes {
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- IssueEvent{node.issue}:
			}

			// issue edit events follow the issue event
			mm.fillIssueEditEvents(ctx, &node)
			// last come the timeline events
			mm.fillTimelineEvents(ctx, &node)
		}
		if !issues.PageInfo.HasNextPage {
			break
		}
		issues, hasIssues = mm.queryIssue(ctx, issues.PageInfo.EndCursor)
	}

	// Second pass: pull-requests. GitHub's PR stream is disjoint from issues
	// (they share the repo's number sequence but `issues` never returns PRs).
	// Sorted UPDATED_AT DESC so we can stop paginating the instant we see a
	// PR older than `since`: catchup syncs with nothing new cost 1 point.
	prs, hasPRs := mm.queryPullRequest(ctx, initialCursor)
prPages:
	for hasPRs {
		for _, node := range prs.Nodes {
			// Because the page is sorted newest-updated first, the first PR
			// older than `since` means every subsequent PR on this page and
			// all following pages is also older → bail on the whole scan.
			if !mm.since.IsZero() && node.UpdatedAt.Before(mm.since) {
				break prPages
			}
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- PrEvent{node.pullRequest}:
			}
			mm.fillPrEditEvents(ctx, &node)
			mm.fillPrTimelineEvents(ctx, &node)
		}
		if !prs.PageInfo.HasNextPage {
			break
		}
		prs, hasPRs = mm.queryPullRequest(ctx, prs.PageInfo.EndCursor)
	}
}

func (mm *importMediator) queryPullRequest(ctx context.Context, cursor githubv4.String) (*pullRequestConnection, bool) {
	vars := newPRVars(mm.owner, mm.project)
	if cursor != "" {
		vars["issueAfter"] = cursor
	}

	query := pullRequestQuery{}
	if err := mm.gh.queryImport(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	noteRateLimit("pullRequests", mm.owner, mm.project, query.RateLimit)
	connection := &query.Repository.PullRequests
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) fillPrEditEvents(ctx context.Context, prNode *pullRequestNode) {
	edits := &prNode.UserContentEdits
	hasEdits := true
	for hasEdits {
		for edit := range reverse(edits.Nodes) {
			if edit.Diff == nil || string(*edit.Diff) == "" {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- PrEditEvent{prId: prNode.pullRequest.Id, userContentEdit: edit}:
			}
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, hasEdits = mm.queryPrEdits(ctx, prNode.pullRequest.Id, edits.PageInfo.EndCursor)
	}
}

func (mm *importMediator) queryPrEdits(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*userContentEditConnection, bool) {
	vars := newIssueEditVars()
	vars["gqlNodeId"] = nid
	if cursor == "" {
		vars["issueEditBefore"] = (*githubv4.String)(nil)
	} else {
		vars["issueEditBefore"] = cursor
	}
	query := prEditQuery{}
	if err := mm.gh.queryImport(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.PullRequest.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) fillPrTimelineEvents(ctx context.Context, prNode *pullRequestNode) {
	items := &prNode.TimelineItems
	hasItems := true
	for hasItems {
		for _, item := range items.Nodes {
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- PrTimelineEvent{prId: prNode.pullRequest.Id, prTimelineItem: item}:
			}
			if item.Typename == "IssueComment" {
				mm.fillCommentEditsPr(ctx, &item)
			}
		}
		if !items.PageInfo.HasNextPage {
			break
		}
		items, hasItems = mm.queryPrTimeline(ctx, prNode.pullRequest.Id, items.PageInfo.EndCursor)
	}
}

func (mm *importMediator) queryPrTimeline(ctx context.Context, nid githubv4.ID, cursor githubv4.String) (*prTimelineItemsConnection, bool) {
	vars := newTimelineVars()
	vars["gqlNodeId"] = nid
	vars["reviewCommentFirst"] = githubv4.Int(NumReviewComments)
	if cursor == "" {
		vars["timelineAfter"] = (*githubv4.String)(nil)
	} else {
		vars["timelineAfter"] = cursor
	}
	query := prTimelineQuery{}
	if err := mm.gh.queryImport(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.PullRequest.TimelineItems
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

// QueryReviewComments fetches a page of review comments for a
// PullRequestReview node id. Returns (nodes, nextCursor, hasNextPage). An
// empty cursor argument fetches the first page.
func (mm *importMediator) QueryReviewComments(ctx context.Context, reviewId githubv4.ID, cursor githubv4.String) ([]pullRequestReviewComment, githubv4.String, bool) {
	vars := varmap{
		"gqlNodeId":          reviewId,
		"reviewCommentFirst": githubv4.Int(NumReviewComments),
	}
	if cursor == "" {
		vars["reviewCommentAfter"] = (*githubv4.String)(nil)
	} else {
		vars["reviewCommentAfter"] = cursor
	}

	query := prReviewCommentsQuery{}
	if err := mm.gh.queryImport(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, "", false
	}
	c := query.Node.PullRequestReview.Comments
	return c.Nodes, c.PageInfo.EndCursor, c.PageInfo.HasNextPage
}

func (mm *importMediator) fillCommentEditsPr(ctx context.Context, item *prTimelineItem) {
	if item.Typename != "IssueComment" {
		return
	}
	comment := &item.IssueComment
	edits := &comment.UserContentEdits
	hasEdits := true
	for hasEdits {
		for edit := range reverse(edits.Nodes) {
			if edit.Diff == nil || string(*edit.Diff) == "" {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- CommentEditEvent{commentId: comment.Id, userContentEdit: edit}:
			}
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, hasEdits = mm.queryCommentEdits(ctx, comment.Id, edits.PageInfo.EndCursor)
	}
}

func (mm *importMediator) fillIssueEditEvents(ctx context.Context, issueNode *issueNode) {
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
			case mm.importEvents <- IssueEditEvent{issueId: issueNode.issue.Id, userContentEdit: edit}:
			}
		}
		if !edits.PageInfo.HasPreviousPage {
			break
		}
		edits, hasEdits = mm.queryIssueEdits(ctx, issueNode.issue.Id, edits.PageInfo.EndCursor)
	}
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
	if err := mm.gh.queryImport(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.Issue.UserContentEdits
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) fillTimelineEvents(ctx context.Context, issueNode *issueNode) {
	items := &issueNode.TimelineItems
	hasItems := true
	for hasItems {
		for _, item := range items.Nodes {
			select {
			case <-ctx.Done():
				return
			case mm.importEvents <- TimelineEvent{issueId: issueNode.issue.Id, timelineItem: item}:
			}
			if item.Typename == "IssueComment" {
				// Issue comments are different than other timeline items in that
				// they may have associated user content edits.
				// Right after the comment we send the comment edits.
				mm.fillCommentEdits(ctx, &item)
			}
		}
		if !items.PageInfo.HasNextPage {
			break
		}
		items, hasItems = mm.queryTimeline(ctx, issueNode.issue.Id, items.PageInfo.EndCursor)
	}
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
	if err := mm.gh.queryImport(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.Issue.TimelineItems
	if len(connection.Nodes) <= 0 {
		return nil, false
	}
	return connection, true
}

func (mm *importMediator) fillCommentEdits(ctx context.Context, item *timelineItem) {
	// Here we are only concerned with timeline items of type issueComment.
	if item.Typename != "IssueComment" {
		return
	}
	// First: setup message handling while submitting GraphQL queries.
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
			case mm.importEvents <- CommentEditEvent{commentId: comment.Id, userContentEdit: edit}:
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
	if err := mm.gh.queryImport(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	connection := &query.Node.IssueComment.UserContentEdits
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
		vars["issueAfter"] = cursor
	}
	query := issueQuery{}
	if err := mm.gh.queryImport(ctx, &query, vars, mm.importEvents); err != nil {
		mm.err = err
		return nil, false
	}
	noteRateLimit("issues", mm.owner, mm.project, query.RateLimit)
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

// varmap is a container for Github API's pagination variables
type varmap map[string]interface{}

func newIssueVars(owner, project string, since time.Time) varmap {
	return varmap{
		"owner":             githubv4.String(owner),
		"name":              githubv4.String(project),
		"issueSince":        githubv4.DateTime{Time: since},
		"issueFirst":        githubv4.Int(NumIssues),
		"issueEditLast":     githubv4.Int(NumIssueEdits),
		"issueEditBefore":   (*githubv4.String)(nil),
		"timelineFirst":     githubv4.Int(NumTimelineItems),
		"timelineAfter":     (*githubv4.String)(nil),
		"commentEditLast":   githubv4.Int(NumCommentEdits),
		"commentEditBefore": (*githubv4.String)(nil),
	}
}

// newPRVars returns the variable set for PR-root queries. It mirrors
// newIssueVars but drops the issue-only filter and adds the variable
// required by nested pullRequestReview.comments.
func newPRVars(owner, project string) varmap {
	return varmap{
		"owner":              githubv4.String(owner),
		"name":               githubv4.String(project),
		"issueFirst":         githubv4.Int(NumIssues),
		"issueAfter":         (*githubv4.String)(nil),
		"issueEditLast":      githubv4.Int(NumIssueEdits),
		"issueEditBefore":   (*githubv4.String)(nil),
		"timelineFirst":      githubv4.Int(NumTimelineItems),
		"timelineAfter":      (*githubv4.String)(nil),
		"commentEditLast":    githubv4.Int(NumCommentEdits),
		"commentEditBefore":  (*githubv4.String)(nil),
		"reviewCommentFirst": githubv4.Int(NumReviewComments),
	}
}

func newIssueEditVars() varmap {
	return varmap{
		"issueEditLast": githubv4.Int(NumIssueEdits),
	}
}

func newTimelineVars() varmap {
	return varmap{
		"timelineFirst":      githubv4.Int(NumTimelineItems),
		"commentEditLast":    githubv4.Int(NumCommentEdits),
		"commentEditBefore":  (*githubv4.String)(nil),
		"reviewCommentFirst": githubv4.Int(NumReviewComments),
	}
}

func newCommentEditVars() varmap {
	return varmap{
		"commentEditLast": githubv4.Int(NumCommentEdits),
	}
}
