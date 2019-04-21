package github

import "github.com/shurcooL/githubv4"

type pageInfo struct {
	EndCursor       githubv4.String
	HasNextPage     bool
	StartCursor     githubv4.String
	HasPreviousPage bool
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

type issueComment struct {
	authorEvent
	Body githubv4.String
	Url  githubv4.URI

	UserContentEdits struct {
		Nodes    []userContentEdit
		PageInfo pageInfo
	} `graphql:"userContentEdits(last: $commentEditLast, before: $commentEditBefore)"`
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

type issueTimeline struct {
	authorEvent
	Title string
	Body  githubv4.String
	Url   githubv4.URI

	Timeline struct {
		Edges []struct {
			Cursor githubv4.String
			Node   timelineItem
		}
		PageInfo pageInfo
	} `graphql:"timeline(first: $timelineFirst, after: $timelineAfter)"`

	UserContentEdits struct {
		Nodes    []userContentEdit
		PageInfo pageInfo
	} `graphql:"userContentEdits(last: $issueEditLast, before: $issueEditBefore)"`
}

type issueEdit struct {
	UserContentEdits struct {
		Nodes    []userContentEdit
		PageInfo pageInfo
	} `graphql:"userContentEdits(last: $issueEditLast, before: $issueEditBefore)"`
}

type issueTimelineQuery struct {
	Repository struct {
		Issues struct {
			Nodes    []issueTimeline
			PageInfo pageInfo
		} `graphql:"issues(first: $issueFirst, after: $issueAfter, orderBy: {field: CREATED_AT, direction: ASC}, filterBy: {since: $issueSince})"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

type issueEditQuery struct {
	Repository struct {
		Issues struct {
			Nodes    []issueEdit
			PageInfo pageInfo
		} `graphql:"issues(first: $issueFirst, after: $issueAfter, orderBy: {field: CREATED_AT, direction: ASC}, filterBy: {since: $issueSince})"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

type commentEditQuery struct {
	Repository struct {
		Issues struct {
			Nodes []struct {
				Timeline struct {
					Nodes []struct {
						IssueComment struct {
							UserContentEdits struct {
								Nodes    []userContentEdit
								PageInfo pageInfo
							} `graphql:"userContentEdits(last: $commentEditLast, before: $commentEditBefore)"`
						} `graphql:"... on IssueComment"`
					}
				} `graphql:"timeline(first: $timelineFirst, after: $timelineAfter)"`
			}
		} `graphql:"issues(first: $issueFirst, after: $issueAfter, orderBy: {field: CREATED_AT, direction: ASC}, filterBy: {since: $issueSince})"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

type userQuery struct {
	User struct {
		Login     githubv4.String
		AvatarUrl githubv4.String
		Name      *githubv4.String
		Email     githubv4.String
	} `graphql:"user(login: $login)"`
}
