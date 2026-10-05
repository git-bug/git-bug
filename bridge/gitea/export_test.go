package gitea

import (
	"context"
	"testing"
	"time"

	gitea "gitea.dev/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
)

func TestExportReportsSkippedIssues(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project"}
	srv := fa.NewServer(t)
	repo, backend := newRoundTripRepo(t)
	storeRoundTripToken(t, repo, srv.URL)
	seedRoundTripBug(t, backend, roundTripBug{Title: "mine", Body: "body"})
	other, err := backend.Identities().NewRaw("Other", "", "other", "", nil, nil)
	require.NoError(t, err)
	_, _, err = backend.Bugs().NewRaw(other, time.Now().Unix(), "theirs", "body", nil, nil)
	require.NoError(t, err)
	_, _, err = backend.Bugs().NewRaw(other, time.Now().Unix(), "elsewhere", "body", nil, map[string]string{
		core.MetaKeyOrigin: target, metaKeyGiteaID: "7", metaKeyGiteaBaseURL: "https://other.example/",
		metaKeyGiteaOwner: "o", metaKeyGiteaProject: "p",
	})
	require.NoError(t, err)

	exporter := (&Gitea{}).NewExporter()
	require.NoError(t, exporter.Init(context.Background(), backend, roundTripConfig(srv.URL)))
	ch, err := exporter.ExportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)
	var reasons []string
	for r := range ch {
		require.NoError(t, r.Err)
		if r.Event == core.ExportEventNothing {
			reasons = append(reasons, r.Reason)
		}
	}
	assert.Len(t, fa.Issues, 1)
	assert.Contains(t, reasons, "no changes by testuser, the owner of the token")
	assert.Contains(t, reasons, "issue belongs to another Gitea repository (https://other.example/o/p)")
}

func TestExportReportsSyncedIssuesAsUpToDate(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project"}
	srv := fa.NewServer(t)
	repo, backend := newRoundTripRepo(t)
	storeRoundTripToken(t, repo, srv.URL)
	seedRoundTripBug(t, backend, roundTripBug{Title: "mine", Body: "body"})
	other, err := backend.Identities().NewRaw("Other", "", "other", "", nil, nil)
	require.NoError(t, err)
	// Imported from this repository and unchanged since.
	b, _, err := backend.Bugs().NewRaw(other, time.Now().Unix(), "synced", "body", nil, map[string]string{
		core.MetaKeyOrigin: target, metaKeyGiteaID: "7", metaKeyGiteaBaseURL: srv.URL,
		metaKeyGiteaOwner: "owner", metaKeyGiteaProject: "project",
	})
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())

	exporter := (&Gitea{}).NewExporter()
	require.NoError(t, exporter.Init(context.Background(), backend, roundTripConfig(srv.URL)))
	ch, err := exporter.ExportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)
	seen := false
	for r := range ch {
		require.NoError(t, r.Err)
		if r.EntityId == b.Id() {
			seen = true
			assert.Equal(t, core.ReasonNothingExported, r.Reason)
		}
	}
	assert.True(t, seen)
}

func TestExportDoesNotSelectIdentityFromOtherInstance(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project"}
	srv := fa.NewServer(t)
	repo, backend := newRoundTripRepo(t)
	storeRoundTripToken(t, repo, srv.URL)

	// Create an identity with matching login "testuser", but scoped to a different instance.
	_, err := backend.Identities().NewRaw(
		"Test User",
		"testuser@example.com",
		"testuser",
		"",
		nil,
		map[string]string{
			metaKeyGiteaLogin:       "testuser",
			metaKeyGiteaScopedLogin: scopedLogin("https://other.example/", "testuser"),
		},
	)
	require.NoError(t, err)

	exporter := (&Gitea{}).NewExporter()
	err = exporter.Init(context.Background(), backend, roundTripConfig(srv.URL))
	assert.ErrorIs(t, err, ErrMissingIdentityToken,
		"exporter must not select an identity scoped to a different instance")
}

func TestExportIncrementalLabels(t *testing.T) {
	upstreamLabel := &gitea.Label{ID: 1, Name: "upstream-label"}
	localLabel := &gitea.Label{ID: 2, Name: "local-label"}
	otherLabel := &gitea.Label{ID: 3, Name: "other-label"}
	fa := &giteatest.FakeAPI{
		Owner:      "owner",
		Project:    "project",
		RepoLabels: []*gitea.Label{upstreamLabel, localLabel, otherLabel},
	}
	srv := fa.NewServer(t)
	repo, backend := newRoundTripRepo(t)
	storeRoundTripToken(t, repo, srv.URL)

	// User identity for exporter
	user, err := backend.Identities().NewRaw(
		"Test User",
		"testuser@example.com",
		"testuser",
		"",
		nil,
		map[string]string{metaKeyGiteaLogin: "testuser"},
	)
	require.NoError(t, err)

	// Pre-create issue 1 on fake API with upstreamLabel
	now := time.Now().UTC()
	issue := &gitea.Issue{
		ID: 1, Index: 1, Title: "mine", Body: "body", State: gitea.StateOpen,
		Poster:  &gitea.User{UserName: "testuser"},
		Labels:  []*gitea.Label{upstreamLabel},
		Created: now, Updated: now,
	}
	fa.Issues = []*gitea.Issue{issue}
	fa.LabelsByIssue = map[int64][]*gitea.Label{1: {upstreamLabel}}

	// Seed local bug that corresponds to issue 1
	b, _, err := backend.Bugs().NewRaw(user, now.Unix(), "mine", "body", nil, map[string]string{
		core.MetaKeyOrigin:  target,
		metaKeyGiteaID:      "1",
		metaKeyGiteaOwner:   "owner",
		metaKeyGiteaProject: "project",
		metaKeyGiteaBaseURL: srv.URL,
	})
	require.NoError(t, err)

	// A different author's label change must not be published along with ours.
	other, err := backend.Identities().NewRaw("Other", "", "other", "", nil, nil)
	require.NoError(t, err)
	_, _, err = b.ChangeLabelsRaw(other, now.Add(time.Minute).Unix(), []string{"other-label"}, nil, nil)
	require.NoError(t, err)
	_, _, err = b.ChangeLabelsRaw(user, now.Add(2*time.Minute).Unix(), []string{"local-label"}, nil, nil)
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())

	exporter := (&Gitea{}).NewExporter()
	require.NoError(t, exporter.Init(context.Background(), backend, roundTripConfig(srv.URL)))
	ch, err := exporter.ExportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)
	for r := range ch {
		require.NoError(t, r.Err)
	}

	// Verify both labels are present on the remote issue
	labels := fa.LabelsByIssue[1]
	require.Len(t, labels, 2)
	names := []string{labels[0].Name, labels[1].Name}
	assert.Contains(t, names, "upstream-label")
	assert.Contains(t, names, "local-label")
	assert.NotContains(t, names, "other-label")
}
