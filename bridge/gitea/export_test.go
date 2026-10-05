package gitea

import (
	"context"
	"testing"
	"time"

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
