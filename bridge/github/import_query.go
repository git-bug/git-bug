package github

import "github.com/shurcooL/githubv4"

type rateLimitQuery struct {
	RateLimit rateLimit
}

// rateLimit mirrors GitHub's RateLimit object. Embed this in any top-level
// query struct to piggyback cost/remaining/reset info on a query we'd make
// anyway — it's free to request and invaluable for diagnosing rate-limit
// starvation.
type rateLimit struct {
	Cost      githubv4.Int
	Remaining githubv4.Int
	Limit     githubv4.Int
	ResetAt   githubv4.DateTime
}

type userQuery struct {
	User user `graphql:"user(login: $login)"`
}

type labelsQuery struct {
	Repository struct {
		Labels struct {
			Nodes []struct {
				ID          string `graphql:"id"`
				Name        string `graphql:"name"`
				Color       string `graphql:"color"`
				Description string `graphql:"description"`
			}
			PageInfo pageInfo
		} `graphql:"labels(first: $first, after: $after)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

type loginQuery struct {
	Viewer struct {
		Login string `graphql:"login"`
	} `graphql:"viewer"`
}

type issueQuery struct {
	Repository struct {
		Issues issueConnection `graphql:"issues(first: $issueFirst, after: $issueAfter, orderBy: {field: CREATED_AT, direction: ASC}, filterBy: {since: $issueSince})"`
	} `graphql:"repository(owner: $owner, name: $name)"`
	RateLimit rateLimit
}

// Pull requests are sorted by UPDATED_AT DESC so catchup syncs can break out
// of pagination as soon as they see a PR older than `since` — GitHub's
// pullRequests connection has no filterBy, so this ordering is the only way
// to avoid walking the entire PR history on every sync.
type pullRequestQuery struct {
	Repository struct {
		PullRequests pullRequestConnection `graphql:"pullRequests(first: $issueFirst, after: $issueAfter, orderBy: {field: UPDATED_AT, direction: DESC})"`
	} `graphql:"repository(owner: $owner, name: $name)"`
	RateLimit rateLimit
}

type prTimelineQuery struct {
	Node struct {
		Typename    githubv4.String `graphql:"__typename"`
		PullRequest struct {
			TimelineItems prTimelineItemsConnection `graphql:"timelineItems(first: $timelineFirst, after: $timelineAfter)"`
		} `graphql:"... on PullRequest"`
	} `graphql:"node(id: $gqlNodeId)"`
}

type prEditQuery struct {
	Node struct {
		Typename    githubv4.String `graphql:"__typename"`
		PullRequest struct {
			UserContentEdits userContentEditConnection `graphql:"userContentEdits(last: $issueEditLast, before: $issueEditBefore)"`
		} `graphql:"... on PullRequest"`
	} `graphql:"node(id: $gqlNodeId)"`
}

// prReviewCommentsQuery fetches a page of review comments for a specific
// PullRequestReview node. Used when a review has more than NumReviewComments
// comments — we follow the cursor.
type prReviewCommentsQuery struct {
	Node struct {
		Typename          githubv4.String `graphql:"__typename"`
		PullRequestReview struct {
			Comments pullRequestReviewCommentConnection `graphql:"comments(first: $reviewCommentFirst, after: $reviewCommentAfter)"`
		} `graphql:"... on PullRequestReview"`
	} `graphql:"node(id: $gqlNodeId)"`
}

type issueEditQuery struct {
	Node struct {
		Typename githubv4.String `graphql:"__typename"`
		Issue    struct {
			UserContentEdits userContentEditConnection `graphql:"userContentEdits(last: $issueEditLast, before: $issueEditBefore)"`
		} `graphql:"... on Issue"`
	} `graphql:"node(id: $gqlNodeId)"`
}

type timelineQuery struct {
	Node struct {
		Typename githubv4.String `graphql:"__typename"`
		Issue    struct {
			TimelineItems timelineItemsConnection `graphql:"timelineItems(first: $timelineFirst, after: $timelineAfter)"`
		} `graphql:"... on Issue"`
	} `graphql:"node(id: $gqlNodeId)"`
}

type commentEditQuery struct {
	Node struct {
		Typename     githubv4.String `graphql:"__typename"`
		IssueComment struct {
			UserContentEdits userContentEditConnection `graphql:"userContentEdits(last: $commentEditLast, before: $commentEditBefore)"`
		} `graphql:"... on IssueComment"`
	} `graphql:"node(id: $gqlNodeId)"`
}

type user struct {
	Login     githubv4.String
	AvatarUrl githubv4.String
	Name      *githubv4.String
}

type issueConnection struct {
	Nodes    []issueNode
	PageInfo pageInfo
}

type issueNode struct {
	issue
	UserContentEdits userContentEditConnection `graphql:"userContentEdits(last: $issueEditLast, before: $issueEditBefore)"`
	TimelineItems    timelineItemsConnection   `graphql:"timelineItems(first: $timelineFirst, after: $timelineAfter)"`
}

type issue struct {
	authorEvent
	Title  githubv4.String
	Number githubv4.Int
	Body   githubv4.String
	Url    githubv4.URI
}

type pullRequestConnection struct {
	Nodes    []pullRequestNode
	PageInfo pageInfo
}

type pullRequestNode struct {
	pullRequest
	UserContentEdits userContentEditConnection `graphql:"userContentEdits(last: $issueEditLast, before: $issueEditBefore)"`
	TimelineItems    prTimelineItemsConnection `graphql:"timelineItems(first: $timelineFirst, after: $timelineAfter)"`
}

type pullRequest struct {
	authorEvent
	Title        githubv4.String
	Number       githubv4.Int
	Body         githubv4.String
	Url          githubv4.URI
	IsDraft      githubv4.Boolean
	BaseRefName  githubv4.String
	HeadRefName  githubv4.String
	HeadRefOid   githubv4.GitObjectID
	Merged       githubv4.Boolean
	MergeCommit  *struct {
		Oid githubv4.GitObjectID
	}
	Closed    githubv4.Boolean
	UpdatedAt githubv4.DateTime
}

type prTimelineItemsConnection struct {
	Nodes    []prTimelineItem
	PageInfo pageInfo
}

// prTimelineItem covers both the issue-compatible timeline events and
// the PR-specific ones (merged, converted-to-draft, ready-for-review,
// reviews).
type prTimelineItem struct {
	Typename githubv4.String `graphql:"__typename"`

	// Issue-shared events
	IssueComment      issueComment         `graphql:"... on IssueComment"`
	LabeledEvent      labeledEvent         `graphql:"... on LabeledEvent"`
	UnlabeledEvent    unlabeledEvent       `graphql:"... on UnlabeledEvent"`
	ClosedEvent       struct{ actorEvent } `graphql:"... on ClosedEvent"`
	ReopenedEvent     struct{ actorEvent } `graphql:"... on ReopenedEvent"`
	RenamedTitleEvent renamedTitleEvent    `graphql:"... on RenamedTitleEvent"`

	// PR-only events
	MergedEvent struct {
		actorEvent
		Commit *struct {
			Oid githubv4.GitObjectID
		}
	} `graphql:"... on MergedEvent"`
	ReadyForReviewEvent struct {
		actorEvent
	} `graphql:"... on ReadyForReviewEvent"`
	ConvertToDraftEvent struct {
		actorEvent
	} `graphql:"... on ConvertToDraftEvent"`
	PullRequestReview pullRequestReview `graphql:"... on PullRequestReview"`
}

// pullRequestReview mirrors GitHub's PullRequestReview node. We fetch the
// first NumReviewComments review comments inline; PRs with more get truncated
// in v1 (pagination can be added later).
type pullRequestReview struct {
	authorEvent
	State  githubv4.PullRequestReviewState
	Body   githubv4.String
	Commit *struct {
		Oid githubv4.GitObjectID
	}
	Comments pullRequestReviewCommentConnection `graphql:"comments(first: $reviewCommentFirst)"`
}

type pullRequestReviewCommentConnection struct {
	Nodes    []pullRequestReviewComment
	PageInfo pageInfo
}

type pullRequestReviewComment struct {
	authorEvent
	Body       githubv4.String
	Path       githubv4.String
	StartLine  *githubv4.Int
	Line       githubv4.Int
	Commit     *struct {
		Oid githubv4.GitObjectID
	}
	ReplyTo *struct {
		Id githubv4.ID
	}
}

type timelineItemsConnection struct {
	Nodes    []timelineItem
	PageInfo pageInfo
}

type userContentEditConnection struct {
	Nodes    []userContentEdit
	PageInfo pageInfo
}

type userContentEdit struct {
	Id        githubv4.ID
	CreatedAt githubv4.DateTime
	UpdatedAt githubv4.DateTime
	EditedAt  githubv4.DateTime
	Editor    *actor
	DeletedAt *githubv4.DateTime
	DeletedBy *actor
	Diff      *githubv4.String
}

type label struct {
	Name githubv4.String
}

type labeledEvent struct {
	actorEvent
	Label label
}

type unlabeledEvent struct {
	actorEvent
	Label label
}

type renamedTitleEvent struct {
	actorEvent
	CurrentTitle  githubv4.String
	PreviousTitle githubv4.String
}

type timelineItem struct {
	Typename githubv4.String `graphql:"__typename"`

	// issue
	IssueComment issueComment `graphql:"... on IssueComment"`

	// Label
	LabeledEvent   labeledEvent   `graphql:"... on LabeledEvent"`
	UnlabeledEvent unlabeledEvent `graphql:"... on UnlabeledEvent"`

	// Status
	ClosedEvent struct {
		actorEvent
		// Url githubv4.URI
	} `graphql:"... on  ClosedEvent"`
	ReopenedEvent struct {
		actorEvent
	} `graphql:"... on  ReopenedEvent"`

	// Title
	RenamedTitleEvent renamedTitleEvent `graphql:"... on RenamedTitleEvent"`
}

type issueComment struct {
	authorEvent // NOTE: contains Id
	Body        githubv4.String
	Url         githubv4.URI

	UserContentEdits userContentEditConnection `graphql:"userContentEdits(last: $commentEditLast, before: $commentEditBefore)"`
}

type userActor struct {
	Name  *githubv4.String
	Email githubv4.String
}

type actor struct {
	Typename  githubv4.String `graphql:"__typename"`
	Login     githubv4.String
	AvatarUrl githubv4.String
	// User      struct {
	// 	Name  *githubv4.String
	// 	Email githubv4.String
	// } `graphql:"... on User"`
	User         userActor `graphql:"... on User"`
	Organization struct {
		Name  *githubv4.String
		Email *githubv4.String
	} `graphql:"... on Organization"`
}

type actorEvent struct {
	Id        githubv4.ID
	CreatedAt githubv4.DateTime
	Actor     *actor
}

type authorEvent struct {
	Id        githubv4.ID
	CreatedAt githubv4.DateTime
	Author    *actor
}

type pageInfo struct {
	EndCursor       githubv4.String
	HasNextPage     bool
	StartCursor     githubv4.String
	HasPreviousPage bool
}
