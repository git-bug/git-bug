package todosrht

import (
	"encoding/json"
	"fmt"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/util/text"
)

const metaKeyRemoteState = "todosrht-remote-state"

// remoteState is the last observed remote content, not the current local
// snapshot. Comparing against it keeps a pull from undoing unpushed edits.
type remoteState struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (state remoteState) normalized() remoteState {
	return remoteState{Subject: text.CleanupOneLine(state.Subject), Body: text.Cleanup(state.Body)}
}

func encodeRemoteState(state remoteState) string {
	raw, _ := json.Marshal(state.normalized()) // A struct containing only strings cannot fail.
	return string(raw)
}

func lastRemoteState(snapshot *bug.Snapshot) (remoteState, error) {
	if raw, ok := remoteStateCheckpoint(snapshot); ok {
		var state remoteState
		if err := json.Unmarshal([]byte(raw), &state); err != nil {
			return remoteState{}, fmt.Errorf("invalid SourceHut remote state: %w", err)
		}
		return state.normalized(), nil
	}

	// Migrate older mappings using only imported/exported operations. Pending
	// local operations must never become the baseline for a remote comparison.
	create := snapshot.Operations[0].(*bug.CreateOperation)
	state := remoteState{Subject: create.Title, Body: create.Message}
	for _, op := range snapshot.Operations[1:] {
		if _, mapped := op.GetMetadata(metaKeyTodoSourceHutId); !mapped {
			continue
		}
		switch op := op.(type) {
		case *bug.SetTitleOperation:
			state.Subject = op.Title
		case *bug.EditCommentOperation:
			if op.Target == create.Id() {
				state.Body = op.Message
			}
		}
	}
	return state.normalized(), nil
}

func remoteStateCheckpoint(snapshot *bug.Snapshot) (string, bool) {
	var latest string
	var found bool
	for _, op := range snapshot.AllOperations() {
		if raw, ok := op.GetMetadata(metaKeyRemoteState); ok {
			latest, found = raw, true
		}
	}
	return latest, found
}

func recordExportedField(b *cache.BugCache, field, value string) error {
	state, err := lastRemoteState(b.Snapshot())
	if err != nil {
		return err
	}
	switch field {
	case "subject":
		state.Subject = value
	case "body":
		state.Body = value
	default:
		return fmt.Errorf("unknown remote ticket field %q", field)
	}
	// Metadata is immutable per operation. Attach each checkpoint to the
	// latest operation instead of trying to overwrite the creation metadata.
	operations := b.Snapshot().Operations
	_, err = b.SetMetadata(operations[len(operations)-1].Id(), map[string]string{
		metaKeyRemoteState: encodeRemoteState(state),
	})
	return err
}
