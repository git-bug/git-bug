package gitea

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	gitea "gitea.dev/sdk"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
	"github.com/git-bug/git-bug/bridge/gitea/iterator"
	"github.com/git-bug/git-bug/entities/common"
)

func TestRetryFetchesIncompleteHistoryBeforeSince(t *testing.T) {
	issue := testIssue()
	issue.Updated = issue.Created.Add(48 * time.Hour)
	comment := testTimelineComment(1, "Older comment")
	comment.Poster = &gitea.User{UserName: "flaky"}
	comment.Created, comment.Updated = issue.Created.Add(time.Hour), issue.Created.Add(time.Hour)
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: []*gitea.Issue{issue},
		TimelineByIssue: map[int64][]*gitea.TimelineComment{1: {comment}}, UserErrLogin: "flaky"}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)
	since := issue.Created.Add(24 * time.Hour)
	require.NotEmpty(t, collectErrors(runImportSince(t, gi, backend, since)))
	require.Len(t, backend.Bugs().AllIds(), 1)
	fa.UserErrLogin = ""
	for pass := 0; pass < 2; pass++ {
		require.Empty(t, collectErrors(runImportSince(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend, since)))
		b := onlyBug(t, backend)
		require.Len(t, b.Snapshot().Comments, 2)
		require.Equal(t, "Older comment", b.Snapshot().Comments[1].Message)
	}
}

func TestNoOpEventsDoNotReplayOverLocalChanges(t *testing.T) {
	issue := testIssue()
	issue.State = gitea.StateClosed
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: []*gitea.Issue{issue}}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)
	require.Empty(t, collectErrors(runImport(t, gi, backend)))
	gi.out = make(chan core.ImportResult, 100)
	b := onlyBug(t, backend)
	_, _, err := b.ChangeLabelsRaw(b.Snapshot().Author, 100, []string{"bug"}, nil, map[string]string{metaKeyGiteaID: "1"})
	require.NoError(t, err)
	status := &iterator.StatusEvent{ID: 7, Closed: true, Poster: issue.Poster, Created: issue.Created.Add(time.Hour)}
	label := &iterator.LabelEvent{ID: 8, Kind: iterator.LabelAdded, Poster: issue.Poster,
		Label: &gitea.Label{ID: 1, Name: "bug"}, UpdatedAt: issue.Created.Add(time.Hour)}
	require.NoError(t, gi.importStatus(t.Context(), backend, b, status))
	require.NoError(t, gi.importLabel(t.Context(), backend, b, label))
	require.NoError(t, b.Commit(t.Context()))
	_, err = b.OpenRaw(b.Snapshot().Author, 200, nil)
	require.NoError(t, err)
	_, _, err = b.ChangeLabelsRaw(b.Snapshot().Author, 200, nil, []string{"bug"}, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	b = onlyBug(t, backend)
	require.NoError(t, gi.importStatus(t.Context(), backend, b, status))
	require.NoError(t, gi.importLabel(t.Context(), backend, b, label))
	require.Equal(t, common.OpenStatus, b.Snapshot().Status)
	require.Empty(t, b.Snapshot().Labels)
}

func TestLabelReconciliationPreservesPendingRemoval(t *testing.T) {
	issue := testIssue()
	issue.State = gitea.StateOpen
	label := &gitea.Label{ID: 1, Name: "bug"}
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: []*gitea.Issue{issue}, Labels: []*gitea.Label{label}}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)
	require.Empty(t, collectErrors(runImport(t, gi, backend)))
	b := onlyBug(t, backend)
	_, _, err := b.ChangeLabelsRaw(b.Snapshot().Author, time.Now().Unix(), nil, []string{"bug"}, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	require.Empty(t, collectErrors(runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)))
	require.Empty(t, onlyBug(t, backend).Snapshot().Labels)
}

func TestConfigurationPinsValidatedToken(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project"}
	fake := fa.NewServer(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token validated" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fake.Config.Handler.ServeHTTP(w, r)
	}))
	defer server.Close()
	repo, backend := newRoundTripRepo(t, t)
	var selected *auth.Token
	for _, value := range []string{"validated", "revoked"} {
		cred := auth.NewToken(target, value)
		cred.SetMetadata(auth.MetaKeyLogin, "testuser")
		cred.SetMetadata(auth.MetaKeyBaseURL, server.URL)
		require.NoError(t, auth.Store(repo, cred))
		if selected == nil {
			selected = cred
		}
	}
	conf, err := (&Gitea{}).Configure(t.Context(), backend, core.BridgeParams{
		URL: server.URL + "/owner/project", CredPrefix: selected.ID().String(),
	}, false)
	require.NoError(t, err)
	require.Equal(t, selected.ID().String(), conf["credential-id"])
	require.NoError(t, (&giteaImporter{}).Init(t.Context(), backend, conf))
	require.NoError(t, (&giteaExporter{}).Init(t.Context(), backend, conf))
}

func TestInteractiveConfigurationUsesVerifiedTokenOwner(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project"}
	fake := fa.NewServer(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/user" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"login":"testuser"}`))
			return
		}
		fake.Config.Handler.ServeHTTP(w, r)
	}))
	defer server.Close()
	_, backend := newRoundTripRepo(t, t)
	input, err := os.CreateTemp(t.TempDir(), "stdin")
	require.NoError(t, err)
	defer input.Close()
	_, err = input.WriteString("test-token\n")
	require.NoError(t, err)
	_, err = input.Seek(0, 0)
	require.NoError(t, err)
	oldStdin := os.Stdin
	os.Stdin = input
	defer func() { os.Stdin = oldStdin }()
	conf, err := (&Gitea{}).Configure(t.Context(), backend, core.BridgeParams{
		URL: server.URL + "/owner/project", Login: "another-valid-user",
	}, true)
	require.NoError(t, err)
	require.Equal(t, "testuser", conf[confKeyDefaultLogin])
	require.NoError(t, (&giteaImporter{}).Init(t.Context(), backend, conf))
}

func TestRepeatedExportRefreshesRenamedLabels(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project"}
	server := fa.NewServer(t)
	repo, backend := newRoundTripRepo(t, t)
	storeRoundTripToken(t, repo, server.URL)
	seedRoundTripBug(t, backend, roundTripBug{Title: "First", Body: "body", Labels: []string{"bug"}})
	exporter := &giteaExporter{}
	conf := roundTripConfig(server.URL)
	require.NoError(t, exporter.Init(t.Context(), backend, conf))
	run := func() {
		events, err := exporter.ExportAll(t.Context(), backend, time.Time{})
		require.NoError(t, err)
		for event := range events {
			require.NoError(t, event.Err)
		}
	}
	run()
	require.Len(t, fa.RepoLabels, 1)
	fa.RepoLabels[0].Name = "renamed"
	b, _, err := backend.Bugs().NewRaw(exporter.exporter, time.Now().Unix(), "Second", "body", nil, nil)
	require.NoError(t, err)
	_, _, err = b.ChangeLabelsRaw(exporter.exporter, time.Now().Unix(), []string{"bug"}, nil, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	run()
	require.Len(t, fa.RepoLabels, 2)
	require.Len(t, fa.Issues[1].Labels, 1)
	require.True(t, strings.EqualFold("bug", fa.Issues[1].Labels[0].Name))
}
