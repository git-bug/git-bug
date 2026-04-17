package github

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	m "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/github/mocks"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/interrupt"
)

// TestGithubPRImport exercises the v1 PR import: fetch PRs after issues, map
// each to a bug with Kind=PR, replay merge / close / draft transitions from
// the PR timeline.
func TestGithubPRImport(t *testing.T) {
	clientMock := &mocks.Client{}
	setupPRExpectations(t, clientMock)
	importer := githubImporter{}
	importer.client = &rateLimitHandlerClient{sc: clientMock}

	repo := repository.CreateGoGitTestRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	events, err := importer.ImportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)
	for e := range events {
		require.NoError(t, e.Err)
	}

	// Three PRs: merged, draft, closed-without-merge.
	require.Len(t, backend.Bugs().AllIds(), 3)

	merged, err := backend.Bugs().ResolveBugCreateMetadata(metaKeyGithubUrl, "https://github.com/marcus/to-himself/pull/1")
	require.NoError(t, err)
	mSnap := merged.Snapshot()
	require.Equal(t, common.PRKind, mSnap.Kind)
	require.Equal(t, common.MergedStatus, mSnap.Status)
	require.Equal(t, "refs/heads/main", mSnap.BaseRef)
	require.Equal(t, "refs/heads/feat1", mSnap.HeadRef)
	require.Equal(t, "aaa111", mSnap.HeadCommit)
	require.Equal(t, "mergecommit1", mSnap.MergeCommit)
	// First op is CreatePR.
	require.Equal(t, common.PRKind, mSnap.Operations[0].(*bug.CreateOperation).Kind)

	draft, err := backend.Bugs().ResolveBugCreateMetadata(metaKeyGithubUrl, "https://github.com/marcus/to-himself/pull/2")
	require.NoError(t, err)
	dSnap := draft.Snapshot()
	require.Equal(t, common.PRKind, dSnap.Kind)
	require.Equal(t, common.DraftStatus, dSnap.Status)

	closed, err := backend.Bugs().ResolveBugCreateMetadata(metaKeyGithubUrl, "https://github.com/marcus/to-himself/pull/3")
	require.NoError(t, err)
	cSnap := closed.Snapshot()
	require.Equal(t, common.PRKind, cSnap.Kind)
	require.Equal(t, common.ClosedStatus, cSnap.Status)
	require.Equal(t, "", cSnap.MergeCommit)
}

func TestGithubPRReviewImport(t *testing.T) {
	clientMock := &mocks.Client{}
	setupPRReviewExpectations(t, clientMock)
	importer := githubImporter{}
	importer.client = &rateLimitHandlerClient{sc: clientMock}

	repo := repository.CreateGoGitTestRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	events, err := importer.ImportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)
	for e := range events {
		require.NoError(t, e.Err)
	}

	require.Len(t, backend.Bugs().AllIds(), 1)
	b, err := backend.Bugs().ResolveBugCreateMetadata(metaKeyGithubUrl, "https://github.com/marcus/to-himself/pull/9")
	require.NoError(t, err)
	snap := b.Snapshot()
	require.Equal(t, common.PRKind, snap.Kind)
	require.Len(t, snap.Reviews, 1)
	require.Equal(t, bug.ReviewApproved, snap.Reviews[0].State)
	require.Equal(t, "LGTM", snap.Reviews[0].Body)
	require.Equal(t, "revcommit", snap.Reviews[0].CommitHash)
	require.Len(t, snap.Reviews[0].Comments, 1)
	rc := snap.Reviews[0].Comments[0]
	require.Equal(t, "nit: naming", rc.Body)
	require.Equal(t, "foo.go", rc.Path)
	require.Equal(t, 42, rc.StartLine)
	require.Equal(t, 42, rc.EndLine)
}

func setupPRReviewExpectations(t *testing.T, mock *mocks.Client) {
	expectEmptyIssueQuery(mock)
	expectPullRequestQueryWithReview(mock)
	expectUserQuery(t, mock)
}

// TestGithubPRReviewCommentsPagination covers the case where a review has
// more inline comments than NumReviewComments. The inline batch reports
// HasNextPage=true; the importer must follow the cursor via a
// prReviewCommentsQuery and import the extra comments too.
func TestGithubPRReviewCommentsPagination(t *testing.T) {
	clientMock := &mocks.Client{}
	setupPRReviewPaginationExpectations(t, clientMock)
	importer := githubImporter{}
	importer.client = &rateLimitHandlerClient{sc: clientMock}

	repo := repository.CreateGoGitTestRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	events, err := importer.ImportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)
	for e := range events {
		require.NoError(t, e.Err)
	}

	b, err := backend.Bugs().ResolveBugCreateMetadata(metaKeyGithubUrl, "https://github.com/marcus/to-himself/pull/10")
	require.NoError(t, err)
	snap := b.Snapshot()
	require.Len(t, snap.Reviews, 1)
	// One comment from the inline batch + one from the follow-up page = 2.
	require.Len(t, snap.Reviews[0].Comments, 2)
	require.Equal(t, "inline-comment", snap.Reviews[0].Comments[0].Body)
	require.Equal(t, "paged-comment", snap.Reviews[0].Comments[1].Body)
}

func setupPRReviewPaginationExpectations(t *testing.T, mock *mocks.Client) {
	expectEmptyIssueQuery(mock)
	expectUserQuery(t, mock)

	mock.On("Query", m.Anything, m.AnythingOfType("*github.prTimelineQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {},
	).Maybe()

	mock.On("Query", m.Anything, m.AnythingOfType("*github.pullRequestQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*pullRequestQuery)
			retVal.Repository.PullRequests.Nodes = []pullRequestNode{
				{
					pullRequest: pullRequest{
						authorEvent: authorEvent{
							Id:     "pr-10",
							Author: &actor{Typename: "User", User: userActor{Name: strPtr("marcus")}},
						},
						Title:       "paged pr",
						Number:      10,
						Body:        "body 10",
						Url:         githubv4.URI{URL: &url.URL{Scheme: "https", Host: "github.com", Path: "marcus/to-himself/pull/10"}},
						BaseRefName: "main",
						HeadRefName: "feat10",
						HeadRefOid:  "commit10",
					},
					TimelineItems: prTimelineItemsConnection{
						Nodes: []prTimelineItem{
							{
								Typename: "PullRequestReview",
								PullRequestReview: pullRequestReview{
									authorEvent: authorEvent{
										Id:     "rev-big",
										Author: &actor{Typename: "User", User: userActor{Name: strPtr("reviewer")}},
									},
									State:  githubv4.PullRequestReviewStateCommented,
									Body:   "batch",
									Commit: &struct{ Oid githubv4.GitObjectID }{Oid: "commit10"},
									Comments: pullRequestReviewCommentConnection{
										Nodes: []pullRequestReviewComment{
											{
												authorEvent: authorEvent{
													Id:     "rc-inline",
													Author: &actor{Typename: "User", User: userActor{Name: strPtr("reviewer")}},
												},
												Body:   "inline-comment",
												Path:   "a.go",
												Line:   5,
												Commit: &struct{ Oid githubv4.GitObjectID }{Oid: "commit10"},
											},
										},
										PageInfo: pageInfo{
											EndCursor:   "cursor-1",
											HasNextPage: true,
										},
									},
								},
							},
						},
					},
				},
			}
		},
	).Once()

	mock.On("Query", m.Anything, m.AnythingOfType("*github.prReviewCommentsQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*prReviewCommentsQuery)
			retVal.Node.PullRequestReview.Comments = pullRequestReviewCommentConnection{
				Nodes: []pullRequestReviewComment{
					{
						authorEvent: authorEvent{
							Id:     "rc-paged",
							Author: &actor{Typename: "User", User: userActor{Name: strPtr("reviewer")}},
						},
						Body:   "paged-comment",
						Path:   "b.go",
						Line:   7,
						Commit: &struct{ Oid githubv4.GitObjectID }{Oid: "commit10"},
					},
				},
				PageInfo: pageInfo{HasNextPage: false},
			}
		},
	).Once()
}

func expectPullRequestQueryWithReview(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.prTimelineQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {},
	).Maybe()

	mock.On("Query", m.Anything, m.AnythingOfType("*github.pullRequestQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*pullRequestQuery)

			startLine := githubv4.Int(42)
			_ = startLine // unused for single-line comments: GitHub sets Line=42, StartLine=nil.

			retVal.Repository.PullRequests.Nodes = []pullRequestNode{
				{
					pullRequest: pullRequest{
						authorEvent: authorEvent{
							Id:     "pr-9",
							Author: &actor{Typename: "User", User: userActor{Name: strPtr("marcus")}},
						},
						Title:       "reviewed pr",
						Number:      9,
						Body:        "body 9",
						Url:         githubv4.URI{URL: &url.URL{Scheme: "https", Host: "github.com", Path: "marcus/to-himself/pull/9"}},
						BaseRefName: "main",
						HeadRefName: "feat9",
						HeadRefOid:  "prcommit9",
					},
					TimelineItems: prTimelineItemsConnection{
						Nodes: []prTimelineItem{
							{
								Typename: "PullRequestReview",
								PullRequestReview: pullRequestReview{
									authorEvent: authorEvent{
										Id:     "rev-1",
										Author: &actor{Typename: "User", User: userActor{Name: strPtr("reviewer")}},
									},
									State: githubv4.PullRequestReviewStateApproved,
									Body:  "LGTM",
									Commit: &struct{ Oid githubv4.GitObjectID }{Oid: "revcommit"},
									Comments: pullRequestReviewCommentConnection{
										Nodes: []pullRequestReviewComment{
											{
												authorEvent: authorEvent{
													Id:     "rc-1",
													Author: &actor{Typename: "User", User: userActor{Name: strPtr("reviewer")}},
												},
												Body:   "nit: naming",
												Path:   "foo.go",
												Line:   42,
												Commit: &struct{ Oid githubv4.GitObjectID }{Oid: "revcommit"},
											},
										},
									},
								},
							},
						},
					},
				},
			}
		},
	).Once()
}

func setupPRExpectations(t *testing.T, mock *mocks.Client) {
	// No issues — the first pass returns nothing so we go straight to PRs.
	expectEmptyIssueQuery(mock)
	expectPullRequestQuery(mock)
	expectUserQuery(t, mock)
}

func expectEmptyIssueQuery(mock *mocks.Client) {
	mock.On("Query", m.Anything, m.AnythingOfType("*github.issueQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			// empty
		},
	)
}

func expectPullRequestQuery(mock *mocks.Client) {
	// Each PR also causes a follow-up timelineItems query; the mock below
	// returns empty timelines so the test focuses on the create-time state.
	mock.On("Query", m.Anything, m.AnythingOfType("*github.prTimelineQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {},
	).Maybe()

	mock.On("Query", m.Anything, m.AnythingOfType("*github.pullRequestQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*pullRequestQuery)
			retVal.Repository.PullRequests.Nodes = []pullRequestNode{
				{
					pullRequest: pullRequest{
						authorEvent: authorEvent{
							Id: "pr-1",
							Author: &actor{
								Typename: "User",
								User: userActor{
									Name:  strPtr("marcus"),
								},
							},
						},
						Title:  "merged pr",
						Number: 1,
						Body:   "body 1",
						Url: githubv4.URI{URL: &url.URL{
							Scheme: "https", Host: "github.com", Path: "marcus/to-himself/pull/1",
						}},
						BaseRefName: "main",
						HeadRefName: "feat1",
						HeadRefOid:  "aaa111",
						Merged:      true,
						MergeCommit: &struct {
							Oid githubv4.GitObjectID
						}{Oid: "mergecommit1"},
						Closed: true,
					},
				},
				{
					pullRequest: pullRequest{
						authorEvent: authorEvent{
							Id:     "pr-2",
							Author: &actor{Typename: "User", User: userActor{Name: strPtr("marcus")}},
						},
						Title:       "draft pr",
						Number:      2,
						Body:        "body 2",
						Url:         githubv4.URI{URL: &url.URL{Scheme: "https", Host: "github.com", Path: "marcus/to-himself/pull/2"}},
						BaseRefName: "main",
						HeadRefName: "feat2",
						HeadRefOid:  "bbb222",
						IsDraft:     true,
					},
				},
				{
					pullRequest: pullRequest{
						authorEvent: authorEvent{
							Id:     "pr-3",
							Author: &actor{Typename: "User", User: userActor{Name: strPtr("marcus")}},
						},
						Title:       "closed pr",
						Number:      3,
						Body:        "body 3",
						Url:         githubv4.URI{URL: &url.URL{Scheme: "https", Host: "github.com", Path: "marcus/to-himself/pull/3"}},
						BaseRefName: "main",
						HeadRefName: "feat3",
						HeadRefOid:  "ccc333",
						Closed:      true,
					},
				},
			}
		},
	).Once()

	// Handle rate-limit retry shape used elsewhere.
	mock.On("Query", m.Anything, m.AnythingOfType("*github.rateLimitQuery"), m.Anything).Return(nil).Run(
		func(args m.Arguments) {
			retVal := args.Get(1).(*rateLimitQuery)
			retVal.RateLimit.ResetAt.Time = time.Now().Add(time.Millisecond * 50)
		},
	).Maybe()

	_ = errors.New // keep import if other setup changes
}

func strPtr(s string) *githubv4.String {
	v := githubv4.String(s)
	return &v
}
