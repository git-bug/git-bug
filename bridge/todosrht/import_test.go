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
	err = backend.SetUserIdentity(userIdentity)
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
		userEntity := struct {
			User
			TypeName string `json:"__typename"`
		}{
			User:     User{Id: 10, CanonicalName: "test-user", Username: "test-user", Email: "test@example.com"},
			TypeName: "User",
		}
		submitterEntity := struct {
			User
			TypeName string `json:"__typename"`
		}{
			User:     User{Id: 11, CanonicalName: "submitter", Username: "submitter", Email: "submitter@example.com"},
			TypeName: "User",
		}

		// Convert submitter to JSON raw message
		submitterJSON, err := json.Marshal(submitterEntity)
		require.NoError(t, err)
		submitterRaw := json.RawMessage(submitterJSON)

		mockClient.MockGetTickets = func(ctx context.Context, trackerName string, cursor *string) ([]Ticket, *string, error) {
			assert.Equal(t, conf[confKeyTrackerName], trackerName)
			return []Ticket{
				{
					Id:        101,
					Created:   Time(time.Now().Add(-24 * time.Hour)),
					Updated:   Time(time.Now()),
					Submitter: &submitterRaw,
					Ref:       "~test-user/test-tracker/101",
					Subject:   "Test Ticket Subject",
					Body:      "Test Ticket Body",
					Status:    TicketStatusReported,
				},
			}, nil, nil // No more pages
		}

		mockClient.MockGetEvents = func(ctx context.Context, trackerName string, ticketID int, cursor *string) ([]Event, *string, error) {
			assert.Equal(t, conf[confKeyTrackerName], trackerName)
			assert.Equal(t, 101, ticketID)

			// Helper to create event changes
			mustMarshal := func(v interface{}) json.RawMessage {
				d, err := json.Marshal(v)
				require.NoError(t, err)
				return d
			}

			userRaw, err := json.Marshal(userEntity)
			require.NoError(t, err)
			userRawPtr := json.RawMessage(userRaw)

			submitterRaw, err := json.Marshal(submitterEntity)
			require.NoError(t, err)
			submitterRawPtr := json.RawMessage(submitterRaw)

			return []Event{
				{
					Id:      1,
					Created: Time(time.Now().Add(-24 * time.Hour)),
					Changes: []json.RawMessage{
						mustMarshal(struct {
							Created
							TypeName string `json:"__typename"`
						}{
							Created: Created{
								EventTypeVal: EventTypeCreated,
								Author:       &submitterRawPtr,
							},
							TypeName: "Created",
						}),
					},
				},
				{
					Id:      2,
					Created: Time(time.Now().Add(-23 * time.Hour)),
					Changes: []json.RawMessage{
						mustMarshal(struct {
							Comment
							TypeName string `json:"__typename"`
						}{
							Comment: Comment{
								EventTypeVal: EventTypeComment,
								Author:       &userRawPtr,
								Text:         "First comment",
							},
							TypeName: "Comment",
						}),
					},
				},
				{
					Id:      3,
					Created: Time(time.Now().Add(-21 * time.Hour)),
					Changes: []json.RawMessage{
						mustMarshal(struct {
							StatusChange
							TypeName string `json:"__typename"`
						}{
							StatusChange: StatusChange{
								EventTypeVal: EventTypeStatusChange,
								Editor:       &userRawPtr,
								NewStatus:    TicketStatusResolved,
							},
							TypeName: "StatusChange",
						}),
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
		userEntity := struct {
			User
			TypeName string `json:"__typename"`
		}{
			User:     User{Id: 10, CanonicalName: "test-user", Username: "test-user", Email: "test@example.com"},
			TypeName: "User",
		}
		submitterEntity := struct {
			User
			TypeName string `json:"__typename"`
		}{
			User:     User{Id: 11, CanonicalName: "submitter", Username: "submitter", Email: "submitter@example.com"},
			TypeName: "User",
		}

		// Convert submitter to JSON raw message
		submitterJSON2, err := json.Marshal(submitterEntity)
		require.NoError(t, err)
		submitterRaw2 := json.RawMessage(submitterJSON2)

		mockClient.MockGetTickets = func(ctx context.Context, trackerName string, cursor *string) ([]Ticket, *string, error) {
			return []Ticket{
				{
					Id:        102,
					Created:   Time(time.Now().Add(-48 * time.Hour)),
					Updated:   Time(time.Now()),
					Submitter: &submitterRaw2,
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
		mockClient.MockGetEvents = func(ctx context.Context, trackerName string, ticketID int, cursor *string) ([]Event, *string, error) {
			assert.Equal(t, conf[confKeyTrackerName], trackerName)

			// Helper to create event changes
			mustMarshal := func(v interface{}) json.RawMessage {
				d, err := json.Marshal(v)
				require.NoError(t, err)
				return d
			}
			userRaw, err := json.Marshal(userEntity)
			require.NoError(t, err)
			userRawPtr := json.RawMessage(userRaw)

			return []Event{
				{
					Id:      4,
					Created: Time(time.Now().Add(-47 * time.Hour)),
					Changes: []json.RawMessage{
						mustMarshal(struct {
							LabelUpdate
							TypeName string `json:"__typename"`
						}{
							LabelUpdate: LabelUpdate{
								EventTypeVal: EventTypeLabelAdded,
								Labeler:      &userRawPtr,
								Label:        Label{Id: 1, Name: "bug"},
							},
							TypeName: "LabelUpdate",
						}),
					},
				},
				{
					Id:      5,
					Created: Time(time.Now().Add(-46 * time.Hour)),
					Changes: []json.RawMessage{
						mustMarshal(struct {
							LabelUpdate
							TypeName string `json:"__typename"`
						}{
							LabelUpdate: LabelUpdate{
								EventTypeVal: EventTypeLabelRemoved,
								Labeler:      &userRawPtr,
								Label:        Label{Id: 1, Name: "bug"},
							},
							TypeName: "LabelUpdate",
						}),
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
