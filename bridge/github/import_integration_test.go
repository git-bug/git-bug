package github

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
	m "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/github/mocks"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository/repotest"
)

// using testify/mock and mockery

var userName = githubv4.String("marcus")
var userEmail = githubv4.String("marcus@rom.com")
var unedited = githubv4.String("unedited")
var edited = githubv4.String("edited")

func TestGithubImporterIntegration(t *testing.T) {
	// mock
	clientMock := &mocks.Client{}
	setupExpectations(t, clientMock)
	importer := githubImporter{}
	importer.client = &rateLimitHandlerClient{sc: clientMock}

	// arrange
	repo := repotest.NewGoGitRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)

	defer backend.Close()
	require.NoError(t, err)

	// act
	events, err := importer.ImportAll(context.Background(), backend, time.Time{})

	// assert
	require.NoError(t, err)
	for e := range events {
		require.NoError(t, e.Err)
	}
	require.Len(t, backend.Bugs().AllIds(), 5)
	require.Len(t, backend.Identities().AllIds(), 2)

	b1, err := backend.Bugs().ResolveBugCreateMetadata(t.Context(), metaKeyGithubUrl, "https://github.com/marcus/to-himself/issues/1")
	require.NoError(t, err)
	ops1 := b1.Snapshot().Operations
	require.Equal(t, "marcus", ops1[0].Author().Name())
	require.Equal(t, "title 1", ops1[0].(*bug.CreateOperation).Title)
	require.Equal(t, "body text 1", ops1[0].(*bug.CreateOperation).Message)

	b3, err := backend.Bugs().ResolveBugCreateMetadata(t.Context(), metaKeyGithubUrl, "https://github.com/marcus/to-himself/issues/3")
	require.NoError(t, err)
	ops3 := b3.Snapshot().Operations
	require.Equal(t, "issue 3 comment 1", ops3[1].(*bug.AddCommentOperation).Message)
	require.Equal(t, "issue 3 comment 2", ops3[2].(*bug.AddCommentOperation).Message)
	require.Equal(t, []common.Label{"bug"}, ops3[3].(*bug.LabelChangeOperation).Added)
	require.Equal(t, "title 3, edit 1", ops3[4].(*bug.SetTitleOperation).Title)

	b4, err := backend.Bugs().ResolveBugCreateMetadata(t.Context(), metaKeyGithubUrl, "https://github.com/marcus/to-himself/issues/4")
	require.NoError(t, err)
	ops4 := b4.Snapshot().Operations
	require.Equal(t, "edited", ops4[1].(*bug.EditCommentOperation).Message)

}

func setupExpectations(t *testing.T, mock *mocks.Client) {
	rateLimitingError(mock)
	expectIssueQuery1(mock)
	expectIssueQuery2(mock)
	expectIssueQuery3(mock)
	expectUserQuery(t, mock)
}

func rateLimitingError(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.issueQuery"), m.Anything).Return(errors.New("API rate limit exceeded")).Once()
	mock.On("Query", m.Anything, m.AnythingOfType("*github.rateLimitQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*rateLimitQuery)
			retVal.RateLimit.ResetAt.Time = time.Now().Add(time.Millisecond * 200)
		},
	).Once()
}

func expectIssueQuery1(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.issueQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*issueQuery)
			retVal.Repository.Issues.Nodes = []issueNode{
				{
					issue: issue{
						authorEvent: authorEvent{
							Id: 1,
							Author: &actor{
								Typename: "User",
								User: userActor{
									Name:  &userName,
									Email: userEmail,
								},
							},
						},
						Title:  "title 1",
						Number: 1,
						Body:   "body text 1",
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/1",
							},
						},
					},
					UserContentEdits: userContentEditConnection{},
					TimelineItems:    timelineItemsConnection{},
				},
				{
					issue: issue{
						authorEvent: authorEvent{
							Id: 2,
							Author: &actor{
								Typename: "User",
								User: userActor{
									Name:  &userName,
									Email: userEmail,
								},
							},
						},
						Title:  "title 2",
						Number: 2,
						Body:   "body text 2",
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/2",
							},
						},
					},
					UserContentEdits: userContentEditConnection{},
					TimelineItems:    timelineItemsConnection{},
				},
			}
			retVal.Repository.Issues.PageInfo = pageInfo{
				EndCursor:   "end-cursor-1",
				HasNextPage: true,
			}
		},
	).Once()
}

func expectIssueQuery2(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.issueQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*issueQuery)
			retVal.Repository.Issues.Nodes = []issueNode{
				{
					issue: issue{
						authorEvent: authorEvent{
							Id: 3,
							Author: &actor{
								Typename: "User",
								User: userActor{
									Name:  &userName,
									Email: userEmail,
								},
							},
						},
						Title:  "title 3",
						Number: 3,
						Body:   "body text 3",
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/3",
							},
						},
					},
					UserContentEdits: userContentEditConnection{},
					TimelineItems: timelineItemsConnection{
						Nodes: []timelineItem{
							{
								Typename: "IssueComment",
								IssueComment: issueComment{
									authorEvent: authorEvent{
										Id: 301,
										Author: &actor{
											Typename: "User",
											User: userActor{
												Name:  &userName,
												Email: userEmail,
											},
										},
									},
									Body: "issue 3 comment 1",
									Url: githubv4.URI{
										URL: &url.URL{
											Scheme: "https",
											Host:   "github.com",
											Path:   "marcus/to-himself/issues/3#issuecomment-1",
										},
									},
									UserContentEdits: userContentEditConnection{},
								},
							},
							{
								Typename: "IssueComment",
								IssueComment: issueComment{
									authorEvent: authorEvent{
										Id: 302,
										Author: &actor{
											Typename: "User",
											User: userActor{
												Name:  &userName,
												Email: userEmail,
											},
										},
									},
									Body: "issue 3 comment 2",
									Url: githubv4.URI{
										URL: &url.URL{
											Scheme: "https",
											Host:   "github.com",
											Path:   "marcus/to-himself/issues/3#issuecomment-2",
										},
									},
									UserContentEdits: userContentEditConnection{},
								},
							},
							{
								Typename: "LabeledEvent",
								LabeledEvent: labeledEvent{
									actorEvent: actorEvent{
										Id: 303,
										Actor: &actor{
											Typename: "User",
											User: userActor{
												Name:  &userName,
												Email: userEmail,
											},
										},
									},
									Label: label{
										Name: "bug",
									},
								},
							},
							{
								Typename: "RenamedTitleEvent",
								RenamedTitleEvent: renamedTitleEvent{
									actorEvent: actorEvent{
										Id: 304,
										Actor: &actor{
											Typename: "User",
											User: userActor{
												Name:  &userName,
												Email: userEmail,
											},
										},
									},
									CurrentTitle: "title 3, edit 1",
								},
							},
						},
						PageInfo: pageInfo{},
					},
				},
				{
					issue: issue{
						authorEvent: authorEvent{
							Id: 4,
							Author: &actor{
								Typename: "User",
								User: userActor{
									Name:  &userName,
									Email: userEmail,
								},
							},
						},
						Title:  "title 4",
						Number: 4,
						Body:   unedited,
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/4",
							},
						},
					},
					UserContentEdits: userContentEditConnection{
						Nodes: []userContentEdit{
							// Github is weird: here the order is reversed chronological
							{
								Id: 402,
								Editor: &actor{
									Typename: "User",
									User: userActor{
										Name:  &userName,
										Email: userEmail,
									},
								},
								Diff: &edited,
							},
							{
								Id: 401,
								Editor: &actor{
									Typename: "User",
									User: userActor{
										Name:  &userName,
										Email: userEmail,
									},
								},
								// Github is weird: whenever an issue has issue edits, then the first item
								// (issue edit) holds the original (unedited) content and the second item
								// (issue edit) holds the (first) edited content.
								Diff: &unedited,
							},
						},
						PageInfo: pageInfo{},
					},
					TimelineItems: timelineItemsConnection{},
				},
			}
			retVal.Repository.Issues.PageInfo = pageInfo{
				EndCursor:   "end-cursor-2",
				HasNextPage: true,
			}
		},
	).Once()
}

func expectIssueQuery3(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.issueQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*issueQuery)
			retVal.Repository.Issues.Nodes = []issueNode{
				{
					issue: issue{
						authorEvent: authorEvent{
							Author: nil,
						},
						Title:  "title 5",
						Number: 5,
						Body:   "body text 5",
						Url: githubv4.URI{
							URL: &url.URL{
								Scheme: "https",
								Host:   "github.com",
								Path:   "marcus/to-himself/issues/5",
							},
						},
					},
					UserContentEdits: userContentEditConnection{},
					TimelineItems:    timelineItemsConnection{},
				},
			}
			retVal.Repository.Issues.PageInfo = pageInfo{}
		},
	).Once()
}

func expectUserQuery(t *testing.T, mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.userQuery"), m.AnythingOfType("map[string]interface {}")).Return(nil).Run(
		func(args m.Arguments) {
			vars := args.Get(2).(map[string]interface{})
			ghost := githubv4.String("ghost")
			require.Equal(t, ghost, vars["login"])

			retVal := args.Get(1).(*userQuery)
			retVal.User.Name = &ghost
			retVal.User.Login = "ghost-login"
		},
	).Once()
}

func countNonMetadataOps(b *cache.BugCache) int {
	count := 0
	for _, op := range b.Snapshot().Operations {
		if _, ok := op.(dag.OperationDoesntChangeSnapshot); !ok {
			count++
		}
	}
	return count
}

func TestGithubImporterDeduplicateExportedOps(t *testing.T) {
	repo := repotest.NewGoGitRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	author, err := backend.Identities().NewRaw(ctx, "marcus", "marcus@rom.com", "marcus", "", nil, map[string]string{
		metaKeyGithubLogin: "marcus",
	})
	require.NoError(t, err)
	require.NoError(t, backend.SetUserIdentity(ctx, author))

	// Create bug imported from GitLab (origin: gitlab) and exported to GitHub (github-id: "issue-1")
	const issueGithubID = "issue-1"
	const issueGithubURL = "https://github.com/marcus/to-himself/issues/1"
	b, createOp, err := backend.Bugs().NewRaw(
		author,
		100,
		"initial title",
		"initial body",
		nil,
		map[string]string{
			"origin":         "gitlab",
			metaKeyGithubId:  issueGithubID,
			metaKeyGithubUrl: issueGithubURL,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, createOp)

	// Simulate exported operations tagged with issue ID (metaKeyGithubId: issueGithubID)
	closeOp, err := b.CloseRaw(author, 101, nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, closeOp.Id(), issueGithubID, issueGithubURL))

	titleOp, err := b.SetTitleRaw(author, 102, "renamed title", nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, titleOp.Id(), issueGithubID, issueGithubURL))

	_, labelAddOp, err := b.ChangeLabelsRaw(author, 103, []string{"bug", "enhancement"}, nil, nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, labelAddOp.Id(), issueGithubID, issueGithubURL))

	_, labelRmOp, err := b.ChangeLabelsRaw(author, 104, nil, []string{"enhancement"}, nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, labelRmOp.Id(), issueGithubID, issueGithubURL))

	require.NoError(t, b.Commit(ctx))
	initialNonMetaOps := countNonMetadataOps(b)

	out := make(chan core.ImportResult, 32)
	importer := &githubImporter{out: out}
	actorMarcus := &actor{
		Typename: "User",
		Login:    "marcus",
		User:     userActor{Name: &userName, Email: userEmail},
	}

	// 1. Echoed events from the export above should be matched to the exported operations (no duplicate operations created)
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "ClosedEvent",
		ClosedEvent: struct{ actorEvent }{
			actorEvent: actorEvent{Id: "evt-close-1", Actor: actorMarcus},
		},
	}))
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "RenamedTitleEvent",
		RenamedTitleEvent: renamedTitleEvent{
			actorEvent:   actorEvent{Id: "evt-title-1", Actor: actorMarcus},
			CurrentTitle: "renamed title",
		},
	}))
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "LabeledEvent",
		LabeledEvent: labeledEvent{
			actorEvent: actorEvent{Id: "evt-label-add-bug", Actor: actorMarcus},
			Label:      label{Name: "bug"},
		},
	}))
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "LabeledEvent",
		LabeledEvent: labeledEvent{
			actorEvent: actorEvent{Id: "evt-label-add-enh", Actor: actorMarcus},
			Label:      label{Name: "enhancement"},
		},
	}))
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "UnlabeledEvent",
		UnlabeledEvent: unlabeledEvent{
			actorEvent: actorEvent{Id: "evt-label-rm-enh", Actor: actorMarcus},
			Label:      label{Name: "enhancement"},
		},
	}))
	require.Equal(t, initialNonMetaOps, countNonMetadataOps(b))

	// 2. Subsequent legitimate remote changes (reopen -> close, new title, re-add enhancement) must be imported
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "ReopenedEvent",
		ReopenedEvent: struct{ actorEvent }{
			actorEvent: actorEvent{Id: "evt-reopen-1", Actor: actorMarcus},
		},
	}))
	require.Equal(t, initialNonMetaOps+1, countNonMetadataOps(b))
	require.Equal(t, common.OpenStatus, b.Snapshot().Status)

	// Re-importing the earlier echoed ClosedEvent ("evt-close-1") must still be skipped!
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "ClosedEvent",
		ClosedEvent: struct{ actorEvent }{
			actorEvent: actorEvent{Id: "evt-close-1", Actor: actorMarcus},
		},
	}))
	require.Equal(t, initialNonMetaOps+1, countNonMetadataOps(b))
	require.Equal(t, common.OpenStatus, b.Snapshot().Status)

	// Now a subsequent ClosedEvent with a new event ID should be imported
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "ClosedEvent",
		ClosedEvent: struct{ actorEvent }{
			actorEvent: actorEvent{Id: "evt-close-2", Actor: actorMarcus},
		},
	}))
	require.Equal(t, initialNonMetaOps+2, countNonMetadataOps(b))
	require.Equal(t, common.ClosedStatus, b.Snapshot().Status)

	// A new title change should be imported
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "RenamedTitleEvent",
		RenamedTitleEvent: renamedTitleEvent{
			actorEvent:   actorEvent{Id: "evt-title-2", Actor: actorMarcus},
			CurrentTitle: "second title",
		},
	}))
	require.Equal(t, initialNonMetaOps+3, countNonMetadataOps(b))
	require.Equal(t, "second title", b.Snapshot().Title)

	// Re-importing the earlier echoed RenamedTitleEvent ("evt-title-1") must still be skipped!
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "RenamedTitleEvent",
		RenamedTitleEvent: renamedTitleEvent{
			actorEvent:   actorEvent{Id: "evt-title-1", Actor: actorMarcus},
			CurrentTitle: "renamed title",
		},
	}))
	require.Equal(t, initialNonMetaOps+3, countNonMetadataOps(b))
	require.Equal(t, "second title", b.Snapshot().Title)

	// Re-adding "enhancement" remotely with a new event ID should be imported (since evt-label-add-enh already claimed labelAddOp)
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "LabeledEvent",
		LabeledEvent: labeledEvent{
			actorEvent: actorEvent{Id: "evt-label-add-enh-2", Actor: actorMarcus},
			Label:      label{Name: "enhancement"},
		},
	}))
	require.Equal(t, initialNonMetaOps+4, countNonMetadataOps(b))

	// Re-importing the earlier echoed UnlabeledEvent ("evt-label-rm-enh") must still be skipped!
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "UnlabeledEvent",
		UnlabeledEvent: unlabeledEvent{
			actorEvent: actorEvent{Id: "evt-label-rm-enh", Actor: actorMarcus},
			Label:      label{Name: "enhancement"},
		},
	}))
	require.Equal(t, initialNonMetaOps+4, countNonMetadataOps(b))

	// 3. Simulate a fresh ImportAll run (new matchedExported map) replaying the entire remote timeline in chronological order
	importer.matchedExported = make(map[string]string)
	fullTimeline := []*timelineItem{
		{
			Typename:    "ClosedEvent",
			ClosedEvent: struct{ actorEvent }{actorEvent: actorEvent{Id: "evt-close-1", Actor: actorMarcus}},
		},
		{
			Typename:          "RenamedTitleEvent",
			RenamedTitleEvent: renamedTitleEvent{actorEvent: actorEvent{Id: "evt-title-1", Actor: actorMarcus}, CurrentTitle: "renamed title"},
		},
		{
			Typename:     "LabeledEvent",
			LabeledEvent: labeledEvent{actorEvent: actorEvent{Id: "evt-label-add-bug", Actor: actorMarcus}, Label: label{Name: "bug"}},
		},
		{
			Typename:     "LabeledEvent",
			LabeledEvent: labeledEvent{actorEvent: actorEvent{Id: "evt-label-add-enh", Actor: actorMarcus}, Label: label{Name: "enhancement"}},
		},
		{
			Typename:       "UnlabeledEvent",
			UnlabeledEvent: unlabeledEvent{actorEvent: actorEvent{Id: "evt-label-rm-enh", Actor: actorMarcus}, Label: label{Name: "enhancement"}},
		},
		{
			Typename:      "ReopenedEvent",
			ReopenedEvent: struct{ actorEvent }{actorEvent: actorEvent{Id: "evt-reopen-1", Actor: actorMarcus}},
		},
		{
			Typename:    "ClosedEvent",
			ClosedEvent: struct{ actorEvent }{actorEvent: actorEvent{Id: "evt-close-2", Actor: actorMarcus}},
		},
		{
			Typename:          "RenamedTitleEvent",
			RenamedTitleEvent: renamedTitleEvent{actorEvent: actorEvent{Id: "evt-title-2", Actor: actorMarcus}, CurrentTitle: "second title"},
		},
		{
			Typename:     "LabeledEvent",
			LabeledEvent: labeledEvent{actorEvent: actorEvent{Id: "evt-label-add-enh-2", Actor: actorMarcus}, Label: label{Name: "enhancement"}},
		},
	}
	for _, item := range fullTimeline {
		require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, item))
	}
	require.Equal(t, initialNonMetaOps+4, countNonMetadataOps(b))

	// 4. Verify ensureIssueEdit and ensureCommentEdit work when multiple exported operations
	// share metaKeyGithubId == issueGithubID (no "Multiple matching operation found" error),
	// and deduplicate exported EditCommentOperation echoes while importing new remote edits.
	editIssueOp, err := b.EditCommentRaw(
		author,
		105,
		entity.CombineIds(b.Id(), createOp.Id()),
		"edited body from git-bug",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, editIssueOp.Id(), issueGithubID, issueGithubURL))

	commentCombinedID, commentOp, err := b.AddCommentRaw(author, 106, "comment 1", nil, nil)
	require.NoError(t, err)
	const commentGithubID = "comment-1"
	const commentGithubURL = "https://github.com/marcus/to-himself/issues/1#issuecomment-1"
	require.NoError(t, markOperationAsExported(ctx, b, commentOp.Id(), commentGithubID, commentGithubURL))

	editCommentOp, err := b.EditCommentRaw(author, 107, commentCombinedID, "comment 1 edited from git-bug", nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, editCommentOp.Id(), commentGithubID, commentGithubURL))

	opsBeforeEdits := countNonMetadataOps(b)

	// Echoed comment creation, issue body edit, and comment edit from git-bug export must be skipped
	require.NoError(t, importer.ensureTimelineItem(ctx, backend, b, &timelineItem{
		Typename: "IssueComment",
		IssueComment: issueComment{
			authorEvent: authorEvent{Id: commentGithubID, Author: actorMarcus},
			Body:        "comment 1 edited from git-bug",
		},
	}))
	echoedIssueBody := githubv4.String("edited body from git-bug")
	require.NoError(t, importer.ensureIssueEdit(ctx, backend, b, issueGithubID, &userContentEdit{
		Id:     "uce-issue-1",
		Editor: actorMarcus,
		Diff:   &echoedIssueBody,
	}))
	echoedCommentBody := githubv4.String("comment 1 edited from git-bug")
	require.NoError(t, importer.ensureCommentEdit(ctx, backend, b, commentGithubID, &userContentEdit{
		Id:     "uce-comment-1",
		Editor: actorMarcus,
		Diff:   &echoedCommentBody,
	}))
	require.Equal(t, opsBeforeEdits, countNonMetadataOps(b))

	// Legitimate remote issue body edit and comment edit must be imported without "Multiple matching operation found"
	remoteIssueBody := githubv4.String("edited body on github")
	require.NoError(t, importer.ensureIssueEdit(ctx, backend, b, issueGithubID, &userContentEdit{
		Id:     "uce-issue-2",
		Editor: actorMarcus,
		Diff:   &remoteIssueBody,
	}))
	require.Equal(t, opsBeforeEdits+1, countNonMetadataOps(b))
	require.Equal(t, "edited body on github", b.Snapshot().Comments[0].Message)

	remoteCommentBody := githubv4.String("comment 1 edited on github")
	require.NoError(t, importer.ensureCommentEdit(ctx, backend, b, commentGithubID, &userContentEdit{
		Id:     "uce-comment-2",
		Editor: actorMarcus,
		Diff:   &remoteCommentBody,
	}))
	require.Equal(t, opsBeforeEdits+2, countNonMetadataOps(b))
	require.Equal(t, "comment 1 edited on github", b.Snapshot().Comments[1].Message)
}
