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
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/repository"
)

func TestImporter(t *testing.T) {
	repo := repository.NewMockRepo()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	conf := core.Configuration{
		confKeyTrackerName:  "test-tracker",
		confKeyBaseUrl:      "https://todo.sr.ht",
		confKeyDefaultLogin: "test-user",
	}

	// Set up identities in repository for both user and submitter
	userIdentity, err := backend.Identities().New("Test User", "test@example.com")
	require.NoError(t, err)
	userIdentity.SetMetadata(auth.MetaKeyLogin, "test-user")
	userIdentity.SetMetadata(metaKeyTodoSourceHutLogin, "test-user")
	err = userIdentity.Commit()
	require.NoError(t, err)

	submitterIdentity, err := backend.Identities().New("Submitter", "submitter@example.com")
	require.NoError(t, err)
	submitterIdentity.SetMetadata(auth.MetaKeyLogin, "submitter")
	submitterIdentity.SetMetadata(metaKeyTodoSourceHutLogin, "submitter")
	err = submitterIdentity.Commit()
	require.NoError(t, err)

	mockClient := &MockClient{}

	importer := &todosrhtImporter{
		conf:   conf,
		client: mockClient,
	}

	mockClient.MockGetTracker = func(ctx context.Context, name string) (*Tracker, error) {
		assert.Equal(t, conf[confKeyTrackerName], name)
		return &Tracker{Id: 1, Name: name}, nil
	}

	t.Run("Import simple ticket with comments", func(t *testing.T) {
		userEntity := User{Id: 10, CanonicalName: "test-user", Username: "test-user", Email: "test@example.com"}
		submitterEntity := User{Id: 11, CanonicalName: "submitter", Username: "submitter", Email: "submitter@example.com"}

		mockClient.MockGetTickets = func(ctx context.Context, trackerID int, cursor *string) ([]Ticket, *string, error) {
			assert.Equal(t, 1, trackerID)
			return []Ticket{
				{
					Id:        101,
					Created:   Time(time.Now().Add(-24 * time.Hour)),
					Updated:   Time(time.Now()),
					Submitter: submitterEntity,
					Ref:       "~test-user/test-tracker/101",
					Subject:   "Test Ticket Subject",
					Body:      "Test Ticket Body",
					Status:    TicketStatusReported,
				},
			}, nil, nil // No more pages
		}

		mockClient.MockGetEvents = func(ctx context.Context, ticketID int, cursor *string) ([]Event, *string, error) {
			assert.Equal(t, 101, ticketID)
			return []Event{
				{
					Id:      1,
					Created: Time(time.Now().Add(-24 * time.Hour)),
					Changes: []EventDetail{
						Created{
							EventTypeVal: EventTypeCreated,
							Author:       submitterEntity,
						},
					},
				},
				{
					Id:      2,
					Created: Time(time.Now().Add(-23 * time.Hour)),
					Changes: []EventDetail{
						Comment{
							EventTypeVal: EventTypeComment,
							Author:       userEntity,
							Text:         "First comment",
						},
					},
				},
				{
					Id:      3,
					Created: Time(time.Now().Add(-21 * time.Hour)),
					Changes: []EventDetail{
						StatusChange{
							EventTypeVal: EventTypeStatusChange,
							Editor:       userEntity,
							NewStatus:    TicketStatusResolved,
						},
					},
				},
			}, nil, nil // No more pages
		}

		events, err := importer.ImportAll(context.Background(), backend, time.Time{})
		require.NoError(t, err)

		var results []core.ImportResult
		for res := range events {
			results = append(results, res)
		}

		// Expected results: 1 Identity, 1 Bug, 1 Comment, 1 StatusChange
		assert.Len(t, results, 4)

		// Verify bug creation
		importedBug, err := backend.Bugs().ResolveBugCreateMetadata(metaKeyTodoSourceHutId, "101")
		require.NoError(t, err)
		assert.Equal(t, "Test Ticket Subject", importedBug.Snapshot().Title)
		assert.Len(t, importedBug.Snapshot().Comments, 2)
		assert.Equal(t, "Test Ticket Body", importedBug.Snapshot().Comments[0].Message)

		// Verify comment
		assert.Len(t, importedBug.Snapshot().Comments, 2)
		assert.Equal(t, "First comment", importedBug.Snapshot().Comments[1].Message)

		// Verify status
		assert.Equal(t, common.ClosedStatus, importedBug.Snapshot().Status)
	})

	t.Run("Import ticket with label changes", func(t *testing.T) {
		userEntity := User{Id: 10, CanonicalName: "test-user", Username: "test-user", Email: "test@example.com"}
		submitterEntity := User{Id: 11, CanonicalName: "submitter", Username: "submitter", Email: "submitter@example.com"}

		mockClient.MockGetTickets = func(ctx context.Context, trackerID int, cursor *string) ([]Ticket, *string, error) {
			return []Ticket{
				{
					Id:        102,
					Created:   Time(time.Now().Add(-48 * time.Hour)),
					Updated:   Time(time.Now()),
					Submitter: submitterEntity,
					Ref:       "~test-user/test-tracker/102",
					Subject:   "Label Test Ticket",
					Body:      "Body",
					Status:    TicketStatusReported,
					Labels: []Label{
						{Id: 1, Name: "bug"},
					},
				},
			}, nil, nil
		}
		mockClient.MockGetEvents = func(ctx context.Context, ticketID int, cursor *string) ([]Event, *string, error) {
			return []Event{
				{
					Id:      4,
					Created: Time(time.Now().Add(-47 * time.Hour)),
					Changes: []EventDetail{
						LabelUpdate{
							EventTypeVal: EventTypeLabelAdded,
							Labeler:      userEntity,
							Label:        Label{Id: 1, Name: "bug"},
						},
					},
				},
				{
					Id:      5,
					Created: Time(time.Now().Add(-46 * time.Hour)),
					Changes: []EventDetail{
						LabelUpdate{
							EventTypeVal: EventTypeLabelRemoved,
							Labeler:      userEntity,
							Label:        Label{Id: 1, Name: "bug"},
						},
					},
				},
			}, nil, nil
		}

		events, err := importer.ImportAll(context.Background(), backend, time.Time{})
		require.NoError(t, err)

		var results []core.ImportResult
		for res := range events {
			results = append(results, res)
		}

		// Verify label changes
		importedBug, err := backend.Bugs().ResolveBugCreateMetadata(metaKeyTodoSourceHutId, "102")
		require.NoError(t, err)
		assert.Len(t, importedBug.Snapshot().Labels, 0) // Labels are added and removed
	})
}
