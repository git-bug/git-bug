package todosrht

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func trackerForTest(owner, name string) *Tracker {
	raw := json.RawMessage(fmt.Sprintf(`{"__typename":"User","canonicalName":%q}`, "~"+owner))
	return &Tracker{Id: 1, Name: name, Owner: &raw}
}

func reviewBackend(t *testing.T) (*cache.RepoCache, *cache.IdentityCache) {
	t.Helper()
	repo := repository.NewMockRepo()
	t.Cleanup(func() { require.NoError(t, repo.Close()) })
	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, backend.Close()) })
	user, err := backend.Identities().New("Tester", "tester@example.org")
	require.NoError(t, err)
	require.NoError(t, backend.SetUserIdentity(t.Context(), user))
	return backend, user
}

func reviewImporter(owner string) *todosrhtImporter {
	return &todosrhtImporter{
		conf: core.Configuration{confKeyBaseUrl: "https://todo.sr.ht", confKeyTrackerName: "~" + owner + "/issues"},
		out:  make(chan core.ImportResult, 100),
	}
}

func reviewTicket() Ticket {
	user := json.RawMessage(`{"__typename":"User","canonicalName":"~tester","username":"tester"}`)
	return Ticket{Id: 1, Subject: "Original", Body: "Body", Submitter: &user,
		Created: Time(time.Unix(1000, 0)), Updated: Time(time.Unix(2000, 0))}
}

func TestCredentialSelectionRequiresExactInstance(t *testing.T) {
	backend, user := reviewBackend(t)
	baseURL := "https://instance-b.sr.ht"
	user.SetMetadata(identityLoginKey(baseURL), "tester")
	require.NoError(t, user.Commit())
	for _, base := range []string{"", "https://instance-a.sr.ht", baseURL} {
		token := auth.NewToken(target, "token-for-"+base)
		token.SetMetadata(auth.MetaKeyLogin, "tester")
		if base != "" {
			token.SetMetadata(auth.MetaKeyBaseURL, base)
		}
		require.NoError(t, auth.Store(backend, token))
	}
	conf := core.Configuration{confKeyBaseUrl: baseURL, confKeyDefaultLogin: "tester"}
	importer := &todosrhtImporter{}
	require.NoError(t, importer.Init(t.Context(), backend, conf))
	require.Equal(t, "token-for-"+baseURL, importer.client.(*TodoSClient).token)
	exporter := &todosrhtExporter{conf: conf, identityClient: make(map[entity.Id]TodosrhtClient)}
	require.NoError(t, exporter.cacheAllClient(t.Context(), backend))
	require.Len(t, exporter.identityClient, 1)
	require.Equal(t, "token-for-"+baseURL, exporter.identityClient[user.Id()].(*TodoSClient).token)
	require.Equal(t, "token-for-"+baseURL, exporter.defaultClient.(*TodoSClient).token)
	conf[confKeyBaseUrl] = "https://instance-c.sr.ht"
	require.ErrorIs(t, importer.Init(t.Context(), backend, conf), ErrMissingCredentials)
	exporter.identityClient = make(map[entity.Id]TodosrhtClient)
	exporter.defaultClient = nil
	require.NoError(t, exporter.cacheAllClient(t.Context(), backend))
	require.Empty(t, exporter.identityClient)
	require.Nil(t, exporter.defaultClient)
}

func TestConfigureQualifiesBareTrackerNames(t *testing.T) {
	backend, _ := reviewBackend(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request GraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		require.Equal(t, "issues", request.Variables["trackerName"])
		require.Contains(t, request.Query, "owner { __typename canonicalName }")
		fmt.Fprint(w, `{"data":{"me":{"tracker":{"id":1,"name":"issues","owner":{"__typename":"User","canonicalName":"~alice"}}}}}`)
	}))
	defer server.Close()
	bridge := &TodoSourceHut{}
	conf, err := bridge.Configure(t.Context(), backend, core.BridgeParams{BaseURL: server.URL, Project: "issues", Login: "alice", TokenRaw: "test-token"}, false)
	require.NoError(t, err)
	require.Equal(t, "~alice/issues", conf[confKeyTrackerName])
}

func TestImporterSeparatesTrackerOwners(t *testing.T) {
	backend, _ := reviewBackend(t)
	for _, owner := range []string{"alice", "bob"} {
		importer := reviewImporter(owner)
		// Older configurations may contain just "issues". Resolve its owner
		// before fetching tickets or matching local metadata.
		importer.conf[confKeyTrackerName] = "issues"
		importer.client = &MockClient{
			MockGetTracker: func(context.Context, string) (*Tracker, error) { return trackerForTest(owner, "issues"), nil },
			MockGetTickets: func(_ context.Context, ref string, _ *string) ([]Ticket, *string, error) {
				require.Equal(t, "~"+owner+"/issues", ref)
				ticket := reviewTicket()
				ticket.Subject = owner
				return []Ticket{ticket}, nil, nil
			},
			MockGetEvents: func(context.Context, string, int, *string) ([]Event, *string, error) { return nil, nil, nil },
		}
		results, err := importer.ImportAll(t.Context(), backend, time.Time{})
		require.NoError(t, err)
		for result := range results {
			require.NoError(t, result.Err)
		}
	}
	require.Len(t, backend.Bugs().AllIds(), 2)
	for _, id := range backend.Bugs().AllIds() {
		b, err := backend.Bugs().Resolve(t.Context(), id)
		require.NoError(t, err)
		ref, _ := b.Snapshot().GetCreateMetadata(metaKeyTodoSourceHutTracker)
		require.Equal(t, "~"+b.Snapshot().Title+"/issues", ref)
	}
}

func TestPullPreservesPendingLocalEdits(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%v", legacy), func(t *testing.T) {
			backend, user := reviewBackend(t)
			importer := reviewImporter("tester")
			ticket := reviewTicket()
			var b *cache.BugCache
			var err error
			if legacy {
				b, _, err = backend.Bugs().NewRaw(user, 1000, ticket.Subject, ticket.Body, nil, map[string]string{
					metaKeyTodoSourceHutId: "1", metaKeyTodoSourceHutTracker: "~tester/issues", metaKeyTodoSourceHutBaseUrl: "https://todo.sr.ht",
				})
			} else {
				b, _, err = importer.ensureIssue(t.Context(), backend, ticket)
			}
			require.NoError(t, err)
			_, err = b.SetTitle(t.Context(), "Local title")
			require.NoError(t, err)
			_, _, err = b.EditCreateComment(t.Context(), "Local body")
			require.NoError(t, err)
			require.NoError(t, b.CommitAsNeeded(t.Context()))
			b, _, err = importer.ensureIssue(t.Context(), backend, ticket)
			require.NoError(t, err)
			require.Equal(t, "Local title", b.Snapshot().Title)
			require.Equal(t, "Local body", b.Snapshot().Comments[0].Message)
			ticket.Subject = "Conflicting remote title"
			_, _, err = importer.ensureIssue(t.Context(), backend, ticket)
			require.ErrorContains(t, err, "conflicting local and remote title")
			require.Equal(t, "Local title", b.Snapshot().Title)
		})
	}
}

func TestRemoteEditsDoNotCreditSubmitter(t *testing.T) {
	backend, _ := reviewBackend(t)
	importer := reviewImporter("tester")
	ticket := reviewTicket()
	b, _, err := importer.ensureIssue(t.Context(), backend, ticket)
	require.NoError(t, err)
	submitter := b.Snapshot().Author.Id()
	require.NoError(t, b.Commit(t.Context()))
	ticket.Subject, ticket.Body = "Remote title", "Remote body"
	b, _, err = importer.ensureIssue(t.Context(), backend, ticket)
	require.NoError(t, err)
	edits := 0
	for _, op := range b.Snapshot().Operations {
		switch op.(type) {
		case *bug.SetTitleOperation, *bug.EditCommentOperation:
			edits++
			require.NotEqual(t, submitter, op.Author().Id())
			require.Equal(t, "SourceHut synchronization", op.Author().Name())
		}
	}
	require.Equal(t, 2, edits)
}

func TestConfigureRejectsCredentialsForOtherServers(t *testing.T) {
	backend, _ := reviewBackend(t)
	for _, targetName := range []string{target, "github"} {
		token := auth.NewToken(targetName, "test-token")
		token.SetMetadata(auth.MetaKeyLogin, "tester")
		token.SetMetadata(auth.MetaKeyBaseURL, "https://instance-a.sr.ht")
		require.NoError(t, auth.Store(backend, token))
		_, err := (&TodoSourceHut{}).Configure(t.Context(), backend, core.BridgeParams{
			BaseURL: "https://instance-b.sr.ht", Project: "issues", CredPrefix: token.ID().String(),
		}, false)
		if targetName == target {
			require.ErrorContains(t, err, "credential is bound to")
		} else {
			require.ErrorContains(t, err, "different bridge target")
		}
	}
}

func TestExportAdvancesRemoteBaseline(t *testing.T) {
	backend, user := reviewBackend(t)
	importer := reviewImporter("tester")
	ticket := reviewTicket()
	b, _, err := importer.ensureIssue(t.Context(), backend, ticket)
	require.NoError(t, err)
	_, err = b.SetTitle(t.Context(), "Exported title")
	require.NoError(t, err)
	_, _, err = b.EditCreateComment(t.Context(), "Exported body")
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded(t.Context()))
	client := &MockClient{MockUpdateTicket: func(_ context.Context, tracker, id int, input UpdateTicketInput) (*Ticket, error) {
		if input.Subject != "" {
			ticket.Subject = input.Subject
		}
		if input.Body != nil {
			ticket.Body = *input.Body
		}
		return &ticket, nil
	}}
	exporter := &todosrhtExporter{conf: importer.conf, tracker: trackerForTest("tester", "issues"),
		identityClient: map[entity.Id]TodosrhtClient{user.Id(): client}, cachedOperationIDs: make(map[entity.Id]string),
	}
	require.NoError(t, exporter.exportBug(t.Context(), b, make(chan core.ExportResult, 100)))
	state, err := lastRemoteState(b.Snapshot())
	require.NoError(t, err)
	require.Equal(t, remoteState{Subject: "Exported title", Body: "Exported body"}, state)
	_, err = b.SetTitle(t.Context(), "Another pending title")
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded(t.Context()))
	b, _, err = importer.ensureIssue(t.Context(), backend, ticket)
	require.NoError(t, err)
	require.Equal(t, "Another pending title", b.Snapshot().Title)
	ticket.Body = "Remote body edit"
	b, _, err = importer.ensureIssue(t.Context(), backend, ticket)
	require.NoError(t, err)
	require.Equal(t, "Another pending title", b.Snapshot().Title)
	require.Equal(t, "Remote body edit", b.Snapshot().Comments[0].Message)
}

func TestRemoteCheckpointsSurviveReload(t *testing.T) {
	repo := repository.NewMockRepo()
	t.Cleanup(func() { require.NoError(t, repo.Close()) })
	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, backend.Close()) })
	user, err := backend.Identities().New("Tester", "tester@example.org")
	require.NoError(t, err)
	require.NoError(t, backend.SetUserIdentity(t.Context(), user))
	b, _, err := backend.Bugs().NewRaw(user, 1000, "Original", "Body", nil, map[string]string{
		metaKeyRemoteState: encodeRemoteState(remoteState{Subject: "Original", Body: "Body"}),
	})
	require.NoError(t, err)
	for _, title := range []string{"First", "Second"} {
		_, err := b.SetTitle(t.Context(), title)
		require.NoError(t, err)
		require.NoError(t, recordExportedField(t.Context(), b, "subject", title))
		require.NoError(t, b.CommitAsNeeded(t.Context()))
		loaded, err := bug.Read(t.Context(), repo, b.Id())
		require.NoError(t, err)
		state, err := lastRemoteState(loaded.Compile())
		require.NoError(t, err)
		require.Equal(t, remoteState{Subject: title, Body: "Body"}, state)
	}
}

func TestImportStopsAtFailedEvent(t *testing.T) {
	backend, _ := reviewBackend(t)
	importer := reviewImporter("tester")
	user := json.RawMessage(`{"__typename":"User","canonicalName":"~tester","username":"tester"}`)
	failed := true
	importer.client = &MockClient{
		MockGetTracker: func(context.Context, string) (*Tracker, error) { return trackerForTest("tester", "issues"), nil },
		MockGetTickets: func(context.Context, string, *string) ([]Ticket, *string, error) {
			return []Ticket{reviewTicket()}, nil, nil
		},
		MockGetEvents: func(context.Context, string, int, *string) ([]Event, *string, error) {
			events := make([]Event, 0, 2)
			for i, status := range []TicketStatus{TicketStatusResolved, TicketStatusReported} {
				var editor *json.RawMessage = &user
				if i == 0 && failed {
					editor = nil
				}
				raw, _ := json.Marshal(map[string]interface{}{"__typename": "StatusChange", "editor": editor, "newStatus": status})
				events = append(events, Event{Id: i + 1, Created: Time(time.Unix(int64(1001+i), 0)), Changes: []json.RawMessage{raw}})
			}
			return events, nil, nil
		},
	}
	for pass := 0; pass < 3; pass++ {
		results, err := importer.ImportAll(t.Context(), backend, time.Time{})
		require.NoError(t, err)
		var errors int
		for result := range results {
			if result.Err != nil {
				errors++
			}
		}
		b, err := backend.Bugs().Resolve(t.Context(), backend.Bugs().AllIds()[0])
		require.NoError(t, err)
		if pass == 0 {
			require.Equal(t, 1, errors)
			require.False(t, hasOperationWithMetadata(b, metaKeyTodoSourceHutId, "2:0"))
		} else {
			require.Zero(t, errors)
			require.True(t, hasOperationWithMetadata(b, metaKeyTodoSourceHutId, "1:0"))
			require.True(t, hasOperationWithMetadata(b, metaKeyTodoSourceHutId, "2:0"))
		}
		require.Equal(t, common.OpenStatus, b.Snapshot().Status)
		failed = false
	}
}

func mappedReviewBug(t *testing.T, backend *cache.RepoCache) *cache.BugCache {
	t.Helper()
	b, _, err := backend.Bugs().New(t.Context(), "Ticket", "Body")
	require.NoError(t, err)
	_, err = b.SetMetadata(t.Context(), b.Snapshot().Operations[0].Id(), map[string]string{
		metaKeyTodoSourceHutId: "1", metaKeyTodoSourceHutTracker: "~tester/issues", metaKeyTodoSourceHutBaseUrl: "https://todo.sr.ht",
	})
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded(t.Context()))
	return b
}

func TestExportStopsAtMissingCredentials(t *testing.T) {
	backend, user := reviewBackend(t)
	other, err := backend.Identities().New("Other", "other@example.org")
	require.NoError(t, err)
	b := mappedReviewBug(t, backend)
	_, err = b.CloseRaw(other, 1001, nil)
	require.NoError(t, err)
	_, err = b.OpenRaw(user, 1002, nil)
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded(t.Context()))
	var statuses []TicketStatus
	makeClient := func(want TicketStatus) *MockClient {
		return &MockClient{MockUpdateTicketStatus: func(_ context.Context, tracker, ticket int, input UpdateStatusInput) (*Event, error) {
			require.Equal(t, want, input.Status)
			statuses = append(statuses, input.Status)
			return &Event{Id: len(statuses), Created: Time(time.Unix(2000, 0))}, nil
		}}
	}
	exporter := &todosrhtExporter{conf: reviewImporter("tester").conf,
		tracker: trackerForTest("tester", "issues"), cachedOperationIDs: make(map[entity.Id]string),
		identityClient: map[entity.Id]TodosrhtClient{user.Id(): makeClient(TicketStatusReported)},
	}
	out := make(chan core.ExportResult, 100)
	require.ErrorIs(t, exporter.exportBug(t.Context(), b, out), ErrMissingCredentials)
	require.Empty(t, statuses)
	exporter.identityClient[other.Id()] = makeClient(TicketStatusResolved)
	require.NoError(t, exporter.exportBug(t.Context(), b, out))
	require.Equal(t, []TicketStatus{TicketStatusResolved, TicketStatusReported}, statuses)
	require.NoError(t, exporter.exportBug(t.Context(), b, out))
	require.Len(t, statuses, 2)
}

func TestLabelReconciliationUsesIDs(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprintf("remove=%v", remove), func(t *testing.T) {
			backend, user := reviewBackend(t)
			b := mappedReviewBug(t, backend)
			var err error
			if remove {
				_, err = b.ForceChangeLabels(t.Context(), nil, []string{"Bug"})
			} else {
				_, err = b.ForceChangeLabels(t.Context(), []string{"Bug"}, nil)
			}
			require.NoError(t, err)
			require.NoError(t, b.CommitAsNeeded(t.Context()))
			var removals int
			client := &MockClient{
				MockGetTicket: func(_ context.Context, tracker string, id int) (*Ticket, error) {
					require.Equal(t, "~tester/issues", tracker)
					return &Ticket{Id: id, Labels: []Label{{Id: 7, Name: "bug"}}}, nil
				},
				MockGetLabels: func(context.Context, string, *string) (*LabelCursor, error) {
					return &LabelCursor{Results: []Label{{Id: 7, Name: "bug"}}}, nil
				},
				MockRemoveLabel: func(_ context.Context, tracker, ticket, label int) (*Event, error) {
					require.Equal(t, 7, label)
					removals++
					return &Event{Id: 10}, nil
				},
			}
			exporter := &todosrhtExporter{conf: reviewImporter("tester").conf,
				tracker: trackerForTest("tester", "issues"), cachedOperationIDs: make(map[entity.Id]string),
				identityClient: map[entity.Id]TodosrhtClient{user.Id(): client},
			}
			require.NoError(t, exporter.exportBug(t.Context(), b, make(chan core.ExportResult, 100)))
			if remove {
				require.Equal(t, 1, removals)
			} else {
				require.Zero(t, removals)
			}
		})
	}
}

func TestIdenticalCommentsAreNotLostAsRevisions(t *testing.T) {
	backend, _ := reviewBackend(t)
	importer := reviewImporter("tester")
	b, _, err := importer.ensureIssue(t.Context(), backend, reviewTicket())
	require.NoError(t, err)
	user := json.RawMessage(`{"__typename":"User","canonicalName":"~tester","username":"tester"}`)
	var events []Event
	for i, text := range []string{"Original", "fixed", "fixed"} {
		var next *Comment
		if i == 0 {
			next = &Comment{Author: &user, Text: "fixed"}
		}
		raw, err := json.Marshal(struct {
			Comment
			TypeName string `json:"__typename"`
		}{Comment: Comment{Author: &user, Text: text, SupersededBy: next}, TypeName: "Comment"})
		require.NoError(t, err)
		events = append(events, Event{Id: i + 1, Created: Time(time.Unix(int64(1001+i), 0)), Changes: []json.RawMessage{raw}})
	}
	for pass := 0; pass < 2; pass++ {
		roots := commentRootsForEvents(events)
		for _, event := range events {
			require.NoError(t, importer.importEvent(t.Context(), backend, b, event, roots))
		}
		require.NoError(t, b.CommitAsNeeded(t.Context()))
		var fixed int
		for _, comment := range b.Snapshot().Comments {
			if comment.Message == "fixed" {
				fixed++
			}
		}
		require.Equal(t, 2, fixed)
		require.True(t, hasOperationWithMetadata(b, metaKeyTodoSourceHutId, "2:0"))
		require.True(t, hasOperationWithMetadata(b, metaKeyTodoSourceHutId, "3:0"))
	}
}
