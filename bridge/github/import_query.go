package github

import "github.com/shurcooL/githubv4"

type rateLimit struct {
	Cost      githubv4.Int
	Limit     githubv4.Int
	NodeCount githubv4.Int
	Remaining githubv4.Int
	ResetAt   githubv4.DateTime
	Used      githubv4.Int
}

type rateLimiter interface {
	rateLimit() rateLimit
}

type userQuery struct {
	RateLimit rateLimit `graphql:"rateLimit(dryRun: $dryRun)"`
	User      user      `graphql:"user(login: $login)"`
}

func (q *userQuery) rateLimit() rateLimit {
	return q.RateLimit
}

type labelsQuery struct {
	//RateLimit rateLimit `graphql:"rateLimit(dryRun: $dryRun)"`
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
	//RateLimit rateLimit `graphql:"rateLimit(dryRun: $dryRun)"`
	Viewer struct {
		Login string `graphql:"login"`
	} `graphql:"viewer"`
}

type issueQuery struct {
	RateLimit  rateLimit `graphql:"rateLimit(dryRun: $dryRun)"`
	Repository struct {
		Issues issueConnection `graphql:"issues(first: $issueFirst, after: $issueAfter, orderBy: {field: CREATED_AT, direction: ASC}, filterBy: {since: $issueSince})"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func (q *issueQuery) rateLimit() rateLimit {
	return q.RateLimit
}

type issueEditQuery struct {
	RateLimit rateLimit `graphql:"rateLimit(dryRun: $dryRun)"`
	Node      struct {
		Typename githubv4.String `graphql:"__typename"`
		Issue    struct {
			UserContentEdits userContentEditConnection `graphql:"userContentEdits(last: $issueEditLast, before: $issueEditBefore)"`
		} `graphql:"... on Issue"`
	} `graphql:"node(id: $gqlNodeId)"`
}

func (q *issueEditQuery) rateLimit() rateLimit {
	return q.RateLimit
}

type timelineQuery struct {
	RateLimit rateLimit `graphql:"rateLimit(dryRun: $dryRun)"`
	Node      struct {
		Typename githubv4.String `graphql:"__typename"`
		Issue    struct {
			TimelineItems timelineItemsConnection `graphql:"timelineItems(first: $timelineFirst, after: $timelineAfter)"`
		} `graphql:"... on Issue"`
	} `graphql:"node(id: $gqlNodeId)"`
}

func (q *timelineQuery) rateLimit() rateLimit {
	return q.RateLimit
}

type commentEditQuery struct {
	RateLimit rateLimit `graphql:"rateLimit(dryRun: $dryRun)"`
	Node      struct {
		Typename     githubv4.String `graphql:"__typename"`
		IssueComment struct {
			UserContentEdits userContentEditConnection `graphql:"userContentEdits(last: $commentEditLast, before: $commentEditBefore)"`
		} `graphql:"... on IssueComment"`
	} `graphql:"node(id: $gqlNodeId)"`
}

func (q *commentEditQuery) rateLimit() rateLimit {
	return q.RateLimit
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

type timelineItem struct {
	Typename githubv4.String `graphql:"__typename"`

	// issue
	IssueComment issueComment `graphql:"... on IssueComment"`

	// Label
	LabeledEvent struct {
		actorEvent
		Label struct {
			// Color githubv4.String
			Name githubv4.String
		}
	} `graphql:"... on LabeledEvent"`
	UnlabeledEvent struct {
		actorEvent
		Label struct {
			// Color githubv4.String
			Name githubv4.String
		}
	} `graphql:"... on UnlabeledEvent"`

	// Status
	ClosedEvent struct {
		actorEvent
		// Url githubv4.URI
	} `graphql:"... on  ClosedEvent"`
	ReopenedEvent struct {
		actorEvent
	} `graphql:"... on  ReopenedEvent"`

	// Title
	RenamedTitleEvent struct {
		actorEvent
		CurrentTitle  githubv4.String
		PreviousTitle githubv4.String
	} `graphql:"... on RenamedTitleEvent"`
}

type issueComment struct {
	authorEvent // NOTE: contains Id
	Body        githubv4.String
	Url         githubv4.URI

	UserContentEdits userContentEditConnection `graphql:"userContentEdits(last: $commentEditLast, before: $commentEditBefore)"`
}

type actor struct {
	Typename  githubv4.String `graphql:"__typename"`
	Login     githubv4.String
	AvatarUrl githubv4.String
	User      struct {
		Name  *githubv4.String
		Email githubv4.String
	} `graphql:"... on User"`
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
