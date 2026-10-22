package github

import (
	"testing"
	"time"

	"github.com/git-bug/git-bug/bridge/github/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestPendingChangeWithoutTokenIsReported(t *testing.T) {
	backend, err := cache.NewRepoCacheNoEvents(repotest.NewGoGitRepo(t, false), nil)
	require.NoError(t, err)
	defer backend.Close()
	author, err := backend.Identities().New("Author", "author@example.org")
	require.NoError(t, err)
	other, err := backend.Identities().New("Other", "other@example.org")
	require.NoError(t, err)
	b, _, err := backend.Bugs().NewRaw(author, time.Now().Unix(), "Mapped", "Body", nil, map[string]string{
		metaKeyGithubId: "issue-node", metaKeyGithubUrl: "https://github.com/owner/project/issues/1",
	})
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	exporter := &githubExporter{conf: core.Configuration{confKeyOwner: "owner", confKeyProject: "project"},
		identityClient: map[entity.Id]*rateLimitHandlerClient{author.Id(): nil}, cachedOperationIDs: map[entity.Id]string{}}
	results := make(chan core.ExportResult, 10)
	exporter.exportBug(t.Context(), b, results)
	require.Equal(t, core.ReasonNothingExported, (<-results).Reason)
	_, _, err = b.AddCommentRaw(other, time.Now().Unix(), "Pending", nil, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	exporter.exportBug(t.Context(), b, results)
	require.Equal(t, core.ReasonPendingNoTokenActor, (<-results).Reason)

	// The credentialed author can still export a body edit while the
	// other author's comment remains pending. Do not call the whole issue
	// unchanged or claim that no credentialed actor changed it.
	require.NoError(t, backend.SetUserIdentity(t.Context(), author))
	_, _, err = b.EditCreateCommentRaw(author, time.Now().Unix(), "New body", nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	client := &mocks.Client{}
	client.On("Mutate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	exporter.identityClient[author.Id()] = &rateLimitHandlerClient{sc: client}
	exporter.exportBug(t.Context(), b, results)
	updated := <-results
	require.NoError(t, updated.Err)
	require.Equal(t, core.ExportEventCommentEdition, updated.Event)
	require.Equal(t, core.ReasonPendingNoTokenActor, (<-results).Reason)
	client.AssertExpectations(t)
}
