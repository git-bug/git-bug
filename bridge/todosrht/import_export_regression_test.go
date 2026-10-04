package todosrht

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
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

func TestExportRejectsUnverifiedTicketID(t *testing.T) {
	repo := repository.NewMockRepo()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	author, err := backend.Identities().New("Tester", "tester@example.com")
	require.NoError(t, err)
	author.SetMetadata(metaKeyTodoSourceHutLogin, "tester")
	require.NoError(t, author.Commit())
	require.NoError(t, backend.SetUserIdentity(author))

	mockClient := &MockClient{}
	exporter := &todosrhtExporter{
		conf: core.Configuration{
			confKeyTrackerName: "~tester/mytracker",
			confKeyBaseUrl:     "https://todo.sr.ht",
		},
		identityClient:     map[entity.Id]TodosrhtClient{author.Id(): mockClient},
		tracker:            &Tracker{Id: 10, Name: "~tester/mytracker"},
		cachedOperationIDs: make(map[entity.Id]string),
	}

	t.Run("ticket ID without tracker metadata is rejected", func(t *testing.T) {
		b, _, err := backend.Bugs().New("Unverified Bug", "Body")
		require.NoError(t, err)
		_, err = b.SetMetadata(b.Snapshot().Operations[0].Id(), map[string]string{
			metaKeyTodoSourceHutId: "42",
		})
		require.NoError(t, err)
		require.NoError(t, b.CommitAsNeeded())

		out := make(chan core.ExportResult, 4)
		err = exporter.exportBug(context.Background(), b, out)
		require.NoError(t, err)
		close(out)

		res := <-out
		assert.Equal(t, core.ExportEventNothing, res.Event)
		assert.Contains(t, res.Reason, "not verified for tracker")
	})

	t.Run("ticket ID with mismatched tracker is rejected", func(t *testing.T) {
		b, _, err := backend.Bugs().New("Mismatched Bug", "Body")
		require.NoError(t, err)
		_, err = b.SetMetadata(b.Snapshot().Operations[0].Id(), map[string]string{
			metaKeyTodoSourceHutId:      "42",
			metaKeyTodoSourceHutTracker: "~other/tracker",
			metaKeyTodoSourceHutBaseUrl: "https://todo.sr.ht",
		})
		require.NoError(t, err)
		require.NoError(t, b.CommitAsNeeded())

		out := make(chan core.ExportResult, 4)
		err = exporter.exportBug(context.Background(), b, out)
		require.NoError(t, err)
		close(out)

		res := <-out
		assert.Equal(t, core.ExportEventNothing, res.Event)
		assert.Contains(t, res.Reason, "tagged with tracker: ~other/tracker")
	})
}

func TestImportReconcilesRemoteEdits(t *testing.T) {
	repo := repository.NewMockRepo()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	author, err := backend.Identities().New("Tester", "tester@example.com")
	require.NoError(t, err)
	author.SetMetadata(metaKeyTodoSourceHutLogin, "tester")
	require.NoError(t, author.Commit())
	require.NoError(t, backend.SetUserIdentity(author))

	b, _, err := backend.Bugs().New("Old Subject", "Old Body")
	require.NoError(t, err)
	_, err = b.SetMetadata(b.Snapshot().Operations[0].Id(), map[string]string{
		metaKeyTodoSourceHutId:      "100",
		metaKeyTodoSourceHutTracker: "~tester/proj",
		metaKeyTodoSourceHutBaseUrl: "https://todo.sr.ht",
	})
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())

	submitterBytes, err := json.Marshal(map[string]string{
		"__typename": "User", "canonicalName": "~tester", "username": "tester", "email": "tester@example.com",
	})
	require.NoError(t, err)
	raw := json.RawMessage(submitterBytes)

	out := make(chan core.ImportResult, 8)
	importer := &todosrhtImporter{
		conf: core.Configuration{
			confKeyTrackerName: "~tester/proj",
			confKeyBaseUrl:     "https://todo.sr.ht",
		},
		out: out,
	}

	updatedTime := Time(time.Now())
	ticket := Ticket{
		Id:        100,
		Subject:   "Updated Subject",
		Body:      "Updated Body",
		Created:   updatedTime,
		Updated:   updatedTime,
		Submitter: &raw,
	}

	got, created, err := importer.ensureIssue(backend, ticket)
	require.NoError(t, err)
	require.False(t, created)
	require.NoError(t, got.CommitAsNeeded())

	// Verify title and body were updated
	snapshot := got.Snapshot()
	assert.Equal(t, "Updated Subject", snapshot.Title)
	assert.Equal(t, "Updated Body", snapshot.Comments[0].Message)

	// Verify edition events were emitted
	close(out)
	var gotTitleEdition, gotCommentEdition bool
	for res := range out {
		if res.Event == core.ImportEventTitleEdition {
			gotTitleEdition = true
		}
		if res.Event == core.ImportEventCommentEdition {
			gotCommentEdition = true
		}
	}
	assert.True(t, gotTitleEdition, "expected TitleEdition event")
	assert.True(t, gotCommentEdition, "expected CommentEdition event")
}

func TestImportMultiChangeEvents(t *testing.T) {
	repo := repository.NewMockRepo()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	author, err := backend.Identities().New("Tester", "tester@example.com")
	require.NoError(t, err)
	author.SetMetadata(metaKeyTodoSourceHutLogin, "tester")
	require.NoError(t, author.Commit())
	require.NoError(t, backend.SetUserIdentity(author))

	b, _, err := backend.Bugs().New("Bug", "Initial message")
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())

	userBytes, err := json.Marshal(map[string]string{
		"__typename": "User", "canonicalName": "~tester", "username": "tester", "email": "tester@example.com",
	})
	require.NoError(t, err)
	rawUser := json.RawMessage(userBytes)

	commentChange, err := json.Marshal(map[string]interface{}{
		"__typename": "Comment",
		"eventType":  EventTypeComment,
		"text":       "A multi-change comment",
		"author":     rawUser,
	})
	require.NoError(t, err)

	statusChange, err := json.Marshal(map[string]interface{}{
		"__typename":    "StatusChange",
		"eventType":     EventTypeStatusChange,
		"oldStatus":     TicketStatusReported,
		"newStatus":     TicketStatusResolved,
		"oldResolution": TicketResolutionUnresolved,
		"newResolution": TicketResolutionFixed,
		"editor":        rawUser,
	})
	require.NoError(t, err)

	event := Event{
		Id:      555,
		Created: Time(time.Now()),
		Changes: []json.RawMessage{commentChange, statusChange},
	}

	importer := &todosrhtImporter{
		conf: core.Configuration{
			confKeyTrackerName: "~tester/proj",
			confKeyBaseUrl:     "https://todo.sr.ht",
		},
		out: make(chan core.ImportResult, 8),
	}

	// First pass: imports both changes
	err = importer.ensureEvent(backend, b, event)
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())

	assert.Len(t, b.Snapshot().Comments, 2)
	assert.Equal(t, common.ClosedStatus, b.Snapshot().Status)

	// Second pass (repeat pull): must succeed without ErrMultipleMatch
	err = importer.ensureEvent(backend, b, event)
	require.NoError(t, err)

	// Ensure no duplicate operations
	assert.Len(t, b.Snapshot().Comments, 2)
}

func TestExportLabelIdempotency(t *testing.T) {
	repo := repository.NewMockRepo()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	author, err := backend.Identities().New("Tester", "tester@example.com")
	require.NoError(t, err)
	author.SetMetadata(metaKeyTodoSourceHutLogin, "tester")
	require.NoError(t, author.Commit())
	require.NoError(t, backend.SetUserIdentity(author))

	b, _, err := backend.Bugs().New("Bug with labels", "Desc")
	require.NoError(t, err)
	_, err = b.SetMetadata(b.Snapshot().Operations[0].Id(), map[string]string{
		metaKeyTodoSourceHutId:      "10",
		metaKeyTodoSourceHutTracker: "~tester/proj",
		metaKeyTodoSourceHutBaseUrl: "https://todo.sr.ht",
	})
	require.NoError(t, err)

	// Add a label change op in git-bug
	_, err = b.ForceChangeLabels([]string{"existing-label"}, nil)
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())

	addLabelCalled := false
	mockClient := &MockClient{
		MockGetTicket: func(ctx context.Context, id int) (*Ticket, error) {
			return &Ticket{
				Id: 10,
				Labels: []Label{
					{Id: 101, Name: "existing-label"},
				},
			}, nil
		},
		MockGetLabels: func(ctx context.Context, trackerName string, cursor *string) (*LabelCursor, error) {
			return &LabelCursor{
				Results: []Label{{Id: 101, Name: "existing-label"}},
			}, nil
		},
		MockAddLabel: func(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error) {
			addLabelCalled = true
			return &Event{Id: 200, Created: Time(time.Now())}, nil
		},
	}

	exporter := &todosrhtExporter{
		conf: core.Configuration{
			confKeyTrackerName: "~tester/proj",
			confKeyBaseUrl:     "https://todo.sr.ht",
		},
		identityClient:     map[entity.Id]TodosrhtClient{author.Id(): mockClient},
		tracker:            &Tracker{Id: 1, Name: "~tester/proj"},
		cachedOperationIDs: make(map[entity.Id]string),
	}

	out := make(chan core.ExportResult, 8)
	err = exporter.exportBug(context.Background(), b, out)
	require.NoError(t, err)

	// Since label was already present on the remote ticket, AddLabel should have been skipped
	assert.False(t, addLabelCalled, "AddLabel should not be called for already-present label")
}

func TestInitTokensWithoutBaseURL(t *testing.T) {
	repo := repository.NewMockRepo()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	// Store token with login but NO base URL (as added by `git bug bridge auth add-token`)
	token := auth.NewToken(target, "my-secret-token")
	token.SetMetadata(auth.MetaKeyLogin, "testuser")
	require.NoError(t, auth.Store(backend, token))

	importer := &todosrhtImporter{}
	conf := core.Configuration{
		core.ConfigKeyTarget: target,
		confKeyBaseUrl:      "https://todo.sr.ht",
		confKeyDefaultLogin: "testuser",
		confKeyTrackerName:  "~testuser/proj",
	}

	err = importer.Init(context.Background(), backend, conf)
	require.NoError(t, err, "importer.Init should accept token without baseURL metadata")
	assert.NotNil(t, importer.client)
}

func TestIdentitiesSeparatedByBaseURL(t *testing.T) {
	repo := repository.NewMockRepo()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	// Create user1 on instance A
	userA, err := backend.Identities().NewRaw(
		"Alice", "alice@example.com", "alice", "", nil,
		map[string]string{
			metaKeyTodoSourceHutLogin:   "alice",
			metaKeyTodoSourceHutBaseUrl: "https://instance-a.sr.ht",
		},
	)
	require.NoError(t, err)

	// Create user1 on instance B
	userB, err := backend.Identities().NewRaw(
		"Alice B", "alice@example.org", "alice", "", nil,
		map[string]string{
			metaKeyTodoSourceHutLogin:   "alice",
			metaKeyTodoSourceHutBaseUrl: "https://instance-b.sr.ht",
		},
	)
	require.NoError(t, err)

	importerA := &todosrhtImporter{
		conf: core.Configuration{
			confKeyBaseUrl: "https://instance-a.sr.ht",
		},
		out: make(chan core.ImportResult, 8),
	}

	userEntity := &User{CanonicalName: "~alice", Username: "alice"}
	matchedA, err := importerA.ensurePerson(backend, userEntity)
	require.NoError(t, err)
	assert.Equal(t, userA.Id(), matchedA.Id(), "should match identity for instance A")

	importerB := &todosrhtImporter{
		conf: core.Configuration{
			confKeyBaseUrl: "https://instance-b.sr.ht",
		},
		out: make(chan core.ImportResult, 8),
	}
	matchedB, err := importerB.ensurePerson(backend, userEntity)
	require.NoError(t, err)
	assert.Equal(t, userB.Id(), matchedB.Id(), "should match identity for instance B")
}
