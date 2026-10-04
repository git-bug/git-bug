package todosrht

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
	"github.com/stretchr/testify/require"
)

func TestImporterReusesExportedBug(t *testing.T) {
	repo := repository.NewMockRepo()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	author, err := backend.Identities().New("Tester", "tester@example.com")
	require.NoError(t, err)
	author.SetMetadata(metaKeyTodoSourceHutLogin, "mcepl")
	require.NoError(t, author.Commit())
	require.NoError(t, backend.SetUserIdentity(author))
	b, _, err := backend.Bugs().New("Local issue", "Body")
	require.NoError(t, err)
	_, err = b.SetMetadata(b.Snapshot().Operations[0].Id(), map[string]string{
		metaKeyTodoSourceHutId:      "1",
		metaKeyTodoSourceHutTracker: "~mcepl/test",
		metaKeyTodoSourceHutBaseUrl: "https://todo.sr.ht",
	})
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())
	submitter, err := json.Marshal(map[string]string{
		"__typename": "User", "canonicalName": "~mcepl", "username": "mcepl", "email": "tester@example.com",
	})
	require.NoError(t, err)
	raw := json.RawMessage(submitter)
	importer := &todosrhtImporter{
		conf: core.Configuration{confKeyTrackerName: "~mcepl/test", confKeyBaseUrl: "https://todo.sr.ht"},
		out:  make(chan core.ImportResult, 8),
	}
	got, created, err := importer.ensureIssue(backend, Ticket{Id: 1, Created: Time(time.Now()), Submitter: &raw})
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, b.Id(), got.Id())
	require.Len(t, backend.Bugs().AllIds(), 1)
}
