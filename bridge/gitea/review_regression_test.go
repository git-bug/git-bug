package gitea

import (
	"context"
	"fmt"
	"testing"
	"time"

	gitea "gitea.dev/sdk"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

type failingIndexRepo struct {
	repository.ClockedRepo
	fail bool
}

func TestImportDoesNotReportUncommittedOperations(t *testing.T) {
	issue := testIssue()
	issue.State = gitea.StateOpen
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: []*gitea.Issue{issue}}
	srv := fa.NewServer(t)
	baseRepo := repository.NewMockRepo()
	repo := &failingIndexRepo{ClockedRepo: baseRepo}
	defer repo.Close()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	defer backend.Close()
	storeRoundTripToken(t, baseRepo, srv.URL)
	importer := setupImporterOnExistingBackend(t, srv.URL, backend)
	require.Empty(t, collectErrors(runImport(t, importer, backend)))
	fa.TimelineByIssue = map[int64][]*gitea.TimelineComment{1: {{ID: 10, Type: "comment", Body: "new", Poster: issue.Poster, Created: issue.Created.Add(time.Hour), Updated: issue.Created.Add(time.Hour)}}}
	repo.fail = true
	results := runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)
	require.NotEmpty(t, collectErrors(results))
	for _, result := range results {
		require.NotEqual(t, core.ImportEventComment, result.Event, "failed persistence must not be reported as a successful import")
	}
	repo.fail = false
}

func (r *failingIndexRepo) GetIndex(name string) (repository.Index, error) {
	if r.fail {
		return nil, fmt.Errorf("index unavailable")
	}
	return r.ClockedRepo.GetIndex(name)
}

func TestExportKeepsRemoteContentAfterRefreshFailure(t *testing.T) {
	for _, comment := range []bool{false, true} {
		t.Run(fmt.Sprintf("comment=%t", comment), func(t *testing.T) {
			fa := &giteatest.FakeAPI{Owner: "owner", Project: "project"}
			srv := fa.NewServer(t)
			baseRepo := repository.NewMockRepo()
			repo := &failingIndexRepo{ClockedRepo: baseRepo}
			defer repo.Close()
			backend, err := cache.NewRepoCacheNoEvents(repo)
			require.NoError(t, err)
			defer backend.Close()
			storeRoundTripToken(t, baseRepo, srv.URL)
			seedRoundTripBug(t, backend, roundTripBug{Title: "mine", Body: "body"})
			exporter := &giteaExporter{}
			require.NoError(t, exporter.Init(context.Background(), backend, roundTripConfig(srv.URL)))
			if comment {
				results, err := exporter.ExportAll(context.Background(), backend, time.Time{})
				require.NoError(t, err)
				for result := range results {
					require.NoError(t, result.Err)
				}
				b := onlyBug(t, backend)
				_, _, err = b.AddCommentRaw(exporter.exporter, time.Now().Unix(), "comment", nil, nil)
				require.NoError(t, err)
				require.NoError(t, b.Commit())
			}
			repo.fail = true
			results, err := exporter.ExportAll(context.Background(), backend, time.Time{})
			require.NoError(t, err)
			var errors int
			for result := range results {
				if result.Err != nil {
					errors++
				}
			}
			require.Positive(t, errors)
			require.Len(t, fa.Issues, 1, "a successfully tracked issue must survive an index refresh failure")
			if comment {
				require.Len(t, fa.TimelineByIssue[1], 1)
			}
			repo.fail = false
			results, err = exporter.ExportAll(context.Background(), backend, time.Time{})
			require.NoError(t, err)
			for result := range results {
				require.NoError(t, result.Err)
			}
			require.Len(t, fa.Issues, 1)
			if comment {
				require.Len(t, fa.TimelineByIssue[1], 1, "retry must not duplicate the persisted comment")
			}
		})
	}
}

func TestImportStatusReconcilesAfterSupersedingRemoteEvent(t *testing.T) {
	issue := testIssue()
	issue.State = gitea.StateOpen
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: []*gitea.Issue{issue}}
	srv := fa.NewServer(t)
	importer, backend := setupImporter(t, srv.URL)
	require.Empty(t, collectErrors(runImport(t, importer, backend)))
	b := onlyBug(t, backend)
	user, err := backend.Identities().New("Local", "local@example.com")
	require.NoError(t, err)
	_, err = b.CloseRaw(user, issue.Created.Add(time.Hour).Unix(), nil)
	require.NoError(t, err)
	_, err = b.OpenRaw(user, issue.Created.Add(2*time.Hour).Unix(), map[string]string{metaKeyGiteaEvent: "reopen:1"})
	require.NoError(t, err)
	require.NoError(t, b.Commit())
	issue.State = gitea.StateClosed
	issue.Updated = issue.Created.Add(3 * time.Hour)
	require.Empty(t, collectErrors(runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)))
	require.Equal(t, "closed", onlyBug(t, backend).Snapshot().Status.String())
}

func TestExportRemovesRenamedImportedLabel(t *testing.T) {
	label := &gitea.Label{ID: 42, Name: "renamed"}
	issue := testIssue()
	issue.Labels = []*gitea.Label{label}
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: []*gitea.Issue{issue}, RepoLabels: []*gitea.Label{label}, LabelsByIssue: map[int64][]*gitea.Label{1: {label}}}
	srv := fa.NewServer(t)
	repo, backend := newRoundTripRepo(t)
	storeRoundTripToken(t, repo, srv.URL)
	seedRoundTripBug(t, backend, roundTripBug{Title: "mine", Body: "body"})
	b := onlyBug(t, backend)
	user := b.Snapshot().Author
	_, err := b.SetMetadataRaw(user, time.Now().Unix(), b.Snapshot().Operations[0].Id(), map[string]string{
		metaKeyGiteaID: "1", metaKeyGiteaBaseURL: srv.URL, metaKeyGiteaOwner: "owner", metaKeyGiteaProject: "project",
	})
	require.NoError(t, err)
	_, _, err = b.ChangeLabelsRaw(user, time.Now().Unix(), []string{"original"}, nil, map[string]string{metaKeyGiteaID: "42"})
	require.NoError(t, err)
	_, _, err = b.ChangeLabelsRaw(user, time.Now().Unix(), nil, []string{"original"}, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit())
	exporter := &giteaExporter{}
	require.NoError(t, exporter.Init(context.Background(), backend, roundTripConfig(srv.URL)))
	results, err := exporter.ExportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)
	for result := range results {
		require.NoError(t, result.Err)
	}
	require.Empty(t, fa.LabelsByIssue[1])
}
