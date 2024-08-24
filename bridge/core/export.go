package core

import (
	"fmt"

	"github.com/git-bug/git-bug/entity"
)

type ExportEvent int

const (
	_ ExportEvent = iota

	// Bug has been exported on the remote tracker
	ExportEventBug
	// Comment has been exported on the remote tracker
	ExportEventComment
	// Comment has been edited on the remote tracker
	ExportEventCommentEdition
	// Bug's status has been changed on the remote tracker
	ExportEventStatusChange
	// Bug's title has been changed on the remote tracker
	ExportEventTitleEdition
	// Bug's labels have been changed on the remote tracker
	ExportEventLabelChange

	// Nothing changed on the bug
	ExportEventNothing

	// Something wrong happened during export that is worth notifying to the user
	// but not severe enough to consider the export a failure.
	ExportEventWarning

	// The export system (web API) has reached a rate limit
	ExportEventRateLimiting

	// Error happened during export
	ExportEventError
)

// ExportResult is an event that is emitted during the export process, to
// allow calling code to report on what is happening, collect metrics or
// display meaningful errors if something went wrong.
type ExportResult struct {
	Err      error
	Event    ExportEvent
	EntityId entity.Id // optional for err, warning
	Reason   string
}

func (er ExportResult) String() string {
	switch er.Event {
	case ExportEventBug:
		return fmt.Sprintf("[%s] new issue: %s", er.EntityId.Human(), er.EntityId)
	case ExportEventComment:
		return fmt.Sprintf("[%s] new comment", er.EntityId.Human())
	case ExportEventCommentEdition:
		return fmt.Sprintf("[%s] updated comment", er.EntityId.Human())
	case ExportEventStatusChange:
		return fmt.Sprintf("[%s] changed status", er.EntityId.Human())
	case ExportEventTitleEdition:
		return fmt.Sprintf("[%s] changed title", er.EntityId.Human())
	case ExportEventLabelChange:
		return fmt.Sprintf("[%s] changed label", er.EntityId.Human())
	case ExportEventNothing:
		if er.EntityId != "" {
			return fmt.Sprintf("no actions taken on entity %s: %s", er.EntityId, er.Reason)
		}
		return fmt.Sprintf("no actions taken: %s", er.Reason)
	case ExportEventError:
		if er.EntityId != "" {
			return fmt.Sprintf("export error on entity %s: %s", er.EntityId, er.Err.Error())
		}
		return fmt.Sprintf("export error: %s", er.Err.Error())
	case ExportEventWarning:
		if er.EntityId != "" {
			return fmt.Sprintf("warning on entity %s: %s", er.EntityId, er.Err.Error())
		}
		return fmt.Sprintf("warning: %s", er.Err.Error())
	case ExportEventRateLimiting:
		return fmt.Sprintf("rate limiting: %s", er.Reason)

	default:
		panic("unknown export result")
	}
}

func NewExportError(err error, entityId entity.Id) ExportResult {
	return ExportResult{
		EntityId: entityId,
		Err:      err,
		Event:    ExportEventError,
	}
}

func NewExportWarning(err error, entityId entity.Id) ExportResult {
	return ExportResult{
		EntityId: entityId,
		Err:      err,
		Event:    ExportEventWarning,
	}
}

func NewExportNothing(entityId entity.Id, reason string) ExportResult {
	return ExportResult{
		EntityId: entityId,
		Reason:   reason,
		Event:    ExportEventNothing,
	}
}

func NewExportBug(entityId entity.Id) ExportResult {
	return ExportResult{
		EntityId: entityId,
		Event:    ExportEventBug,
	}
}

func NewExportComment(entityId entity.Id) ExportResult {
	return ExportResult{
		EntityId: entityId,
		Event:    ExportEventComment,
	}
}

func NewExportCommentEdition(entityId entity.Id) ExportResult {
	return ExportResult{
		EntityId: entityId,
		Event:    ExportEventCommentEdition,
	}
}

func NewExportStatusChange(entityId entity.Id) ExportResult {
	return ExportResult{
		EntityId: entityId,
		Event:    ExportEventStatusChange,
	}
}

func NewExportLabelChange(entityId entity.Id) ExportResult {
	return ExportResult{
		EntityId: entityId,
		Event:    ExportEventLabelChange,
	}
}

func NewExportTitleEdition(entityId entity.Id) ExportResult {
	return ExportResult{
		EntityId: entityId,
		Event:    ExportEventTitleEdition,
	}
}

func NewExportRateLimiting(msg string) ExportResult {
	return ExportResult{
		Reason: msg,
		Event:  ExportEventRateLimiting,
	}
}
