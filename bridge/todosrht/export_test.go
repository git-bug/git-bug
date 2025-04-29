package todosrht

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func TestExporter(t *testing.T) {
	repo := repository.NewMockRepo()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	conf := core.Configuration{
		confKeyTrackerName:  "test-tracker",
		confKeyBaseUrl:      "https://todo.sr.ht",
		confKeyDefaultLogin: "test-user",
	}

	// Set the user identity in the backend first
	author, err := backend.Identities().New("Test User", "test@example.com")
	require.NoError(t, err)

	// Set metadata on identity for cacheAllClient to find it
	author.SetMetadata(auth.MetaKeyLogin, "test-user")
	author.SetMetadata(metaKeyTodoSourceHutLogin, "test-user")
	err = author.Commit()
	require.NoError(t, err)

	// Store a token for author so that Init() can find it
	token := auth.NewToken(target, "test-token")
	token.SetMetadata(auth.MetaKeyLogin, "test-user")
	token.SetMetadata(auth.MetaKeyBaseURL, "https://todo.sr.ht")
	err = auth.Store(repo, token)
	require.NoError(t, err)

	// Set the user identity in the backend
	err = backend.SetUserIdentity(author)
	require.NoError(t, err)

	mockClient := &MockClient{
		MockGetTracker: func(ctx context.Context, name string) (*Tracker, error) {
			return &Tracker{Id: 1, Name: name}, nil
		},
		MockCreateTicket: func(ctx context.Context, trackerID int, input SubmitTicketInput) (*Ticket, error) {
			return &Ticket{Id: 1, Subject: input.Subject}, nil
		},
		MockCreateComment: func(ctx context.Context, trackerID, ticketID int, input SubmitCommentInput) (*Event, error) {
			return &Event{Id: 1}, nil
		},
		MockUpdateTicketStatus: func(ctx context.Context, trackerID, ticketID int, input UpdateStatusInput) (*Event, error) {
			return &Event{Id: 1}, nil
		},
		MockAddLabel: func(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error) {
			return &Event{Id: 1}, nil
		},
		MockRemoveLabel: func(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error) {
			return &Event{Id: 1}, nil
		},
		MockGetLabels: func(ctx context.Context, trackerID int, cursor *string) (*LabelCursor, error) {
			return &LabelCursor{Results: []Label{{Id: 1, Name: "bug"}}}, nil
		},
	}

	exporter := &todosrhtExporter{
		conf:               conf,
		identityClient:     map[entity.Id]TodosrhtClient{author.Id(): mockClient},
		tracker:            &Tracker{Id: 1, Name: "test-tracker"},
		cachedOperationIDs: make(map[entity.Id]string),
	}

	t.Run("Create and export a simple bug", func(t *testing.T) {
		_, _, err := backend.Bugs().New("Simple Bug", "Description of simple bug")
		require.NoError(t, err)

		events, err := exporter.ExportAll(context.Background(), backend, time.Time{})
		require.NoError(t, err)

		for res := range events {
			assert.NoError(t, res.Err)
		}
	})

	t.Run("Add a comment to an existing bug", func(t *testing.T) {
		bugCache, _, err := backend.Bugs().New("Bug with comment", "Description")
		require.NoError(t, err)

		_, _, err = bugCache.AddComment("A first comment")
		require.NoError(t, err)

		events, err := exporter.ExportAll(context.Background(), backend, time.Time{})
		require.NoError(t, err)

		for res := range events {
			assert.NoError(t, res.Err)
		}
	})

	t.Run("Change bug status", func(t *testing.T) {
		bugCache, _, err := backend.Bugs().New("Bug to close", "Description")
		require.NoError(t, err)

		_, err = bugCache.Close()
		require.NoError(t, err)

		events, err := exporter.ExportAll(context.Background(), backend, time.Time{})
		require.NoError(t, err)

		for res := range events {
			assert.NoError(t, res.Err)
		}
	})

	t.Run("Add and remove labels", func(t *testing.T) {
		bugCache, _, err := backend.Bugs().New("Bug with labels", "Description")
		require.NoError(t, err)

		_, _, err = bugCache.ChangeLabels([]string{"bug"}, nil)
		require.NoError(t, err)
		_, _, err = bugCache.ChangeLabels(nil, []string{"bug"})
		require.NoError(t, err)

		events, err := exporter.ExportAll(context.Background(), backend, time.Time{})
		require.NoError(t, err)

		for res := range events {
			assert.NoError(t, res.Err)
		}
	})
}
