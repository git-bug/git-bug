package core

import (
	"fmt"
	"strings"

	"github.com/git-bug/git-bug/entity"
)

type ImportEvent int

const (
	_ ImportEvent = iota

	// Bug has been created
	ImportEventBug
	// Comment has been created
	ImportEventComment
	// Comment has been edited
	ImportEventCommentEdition
	// Bug's status has changed
	ImportEventStatusChange
	// Bug's title has changed
	ImportEventTitleEdition
	// Bug's labels changed
	ImportEventLabelChange
	// Nothing happened on a Bug
	ImportEventNothing

	// Identity has been created
	ImportEventIdentity

	// Something wrong happened during import that is worth notifying to the user
	// but not severe enough to consider the import a failure.
	ImportEventWarning

	// The import system (web API) has reached the rate limit
	ImportEventRateLimiting

	// Error happened during import
	ImportEventError
)

// ImportResult is an event that is emitted during the import process, to
// allow calling code to report on what is happening, collect metrics or
// display meaningful errors if something went wrong.
type ImportResult struct {
	Err         error
	Event       ImportEvent
	EntityId    entity.Id         // optional for err, warnings
	OperationId entity.Id         // optional
	ComponentId entity.CombinedId // optional
	Reason      string
}

func (er ImportResult) String() string {
	switch er.Event {
	case ImportEventBug:
		return fmt.Sprintf("[%s] new issue: %s", er.EntityId.Human(), er.EntityId)
	case ImportEventComment:
		return fmt.Sprintf("[%s] new comment: %s", er.EntityId.Human(), er.ComponentId)
	case ImportEventCommentEdition:
		return fmt.Sprintf("[%s] updated comment: %s", er.EntityId.Human(), er.ComponentId)
	case ImportEventStatusChange:
		return fmt.Sprintf("[%s] changed status with op: %s", er.EntityId.Human(), er.OperationId)
	case ImportEventTitleEdition:
		return fmt.Sprintf("[%s] changed title with op: %s", er.EntityId.Human(), er.OperationId)
	case ImportEventLabelChange:
		return fmt.Sprintf("[%s] changed label with op: %s", er.EntityId.Human(), er.OperationId)
	case ImportEventIdentity:
		return fmt.Sprintf("[%s] new identity: %s", er.EntityId.Human(), er.EntityId)
	case ImportEventNothing:
		if er.EntityId != "" {
			return fmt.Sprintf("no action taken on entity %s: %s", er.EntityId, er.Reason)
		}
		return fmt.Sprintf("no action taken: %s", er.Reason)
	case ImportEventError:
		if er.EntityId != "" {
			return fmt.Sprintf("import error on entity %s: %s", er.EntityId, er.Err.Error())
		}
		return fmt.Sprintf("import error: %s", er.Err.Error())
	case ImportEventWarning:
		parts := make([]string, 0, 4)
		parts = append(parts, "warning:")
		if er.EntityId != "" {
			parts = append(parts, fmt.Sprintf("on entity %s", er.EntityId))
		}
		if er.Reason != "" {
			parts = append(parts, fmt.Sprintf("reason: %s", er.Reason))
		}
		if er.Err != nil {
			parts = append(parts, fmt.Sprintf("err: %s", er.Err))
		}
		return strings.Join(parts, " ")
	case ImportEventRateLimiting:
		return fmt.Sprintf("rate limiting: %s", er.Reason)

	default:
		panic("unknown import result")
	}
}

func NewImportError(err error, entityId entity.Id) ImportResult {
	return ImportResult{
		Err:      err,
		EntityId: entityId,
		Event:    ImportEventError,
	}
}

func NewImportWarning(err error, entityId entity.Id) ImportResult {
	return ImportResult{
		Err:      err,
		EntityId: entityId,
		Event:    ImportEventWarning,
	}
}

func NewImportNothing(entityId entity.Id, reason string) ImportResult {
	return ImportResult{
		EntityId: entityId,
		Reason:   reason,
		Event:    ImportEventNothing,
	}
}

func NewImportBug(entityId entity.Id) ImportResult {
	return ImportResult{
		EntityId: entityId,
		Event:    ImportEventBug,
	}
}

func NewImportComment(entityId entity.Id, commentId entity.CombinedId) ImportResult {
	return ImportResult{
		EntityId:    entityId,
		ComponentId: commentId,
		Event:       ImportEventComment,
	}
}

func NewImportCommentEdition(entityId entity.Id, commentId entity.CombinedId) ImportResult {
	return ImportResult{
		EntityId:    entityId,
		ComponentId: commentId,
		Event:       ImportEventCommentEdition,
	}
}

func NewImportStatusChange(entityId entity.Id, opId entity.Id) ImportResult {
	return ImportResult{
		EntityId:    entityId,
		OperationId: opId,
		Event:       ImportEventStatusChange,
	}
}

func NewImportLabelChange(entityId entity.Id, opId entity.Id) ImportResult {
	return ImportResult{
		EntityId:    entityId,
		OperationId: opId,
		Event:       ImportEventLabelChange,
	}
}

func NewImportTitleEdition(entityId entity.Id, opId entity.Id) ImportResult {
	return ImportResult{
		EntityId:    entityId,
		OperationId: opId,
		Event:       ImportEventTitleEdition,
	}
}

func NewImportIdentity(entityId entity.Id) ImportResult {
	return ImportResult{
		EntityId: entityId,
		Event:    ImportEventIdentity,
	}
}

func NewImportRateLimiting(msg string) ImportResult {
	return ImportResult{
		Reason: msg,
		Event:  ImportEventRateLimiting,
	}
}
