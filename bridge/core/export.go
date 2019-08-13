package core

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
)

type ExportEvent int

const (
	_ ExportEvent = iota
	ExportEventBug
	ExportEventComment
	ExportEventCommentEdition
	ExportEventStatusChange
	ExportEventTitleEdition
	ExportEventLabelChange
	ExportEventNothing
)

// ExportResult is an event that is emitted during the export process, to
// allow calling code to report on what is happening, collect metrics or
// display meaningful errors if something went wrong.
type ExportResult struct {
	Err    error
	Event  ExportEvent
	ID     entity.Id
	Reason string
}

func (er ExportResult) String() string {
	switch er.Event {
	case ExportEventBug:
		return "new issue"
	case ExportEventComment:
		return "new comment"
	case ExportEventCommentEdition:
		return "updated comment"
	case ExportEventStatusChange:
		return "changed status"
	case ExportEventTitleEdition:
		return "changed title"
	case ExportEventLabelChange:
		return "changed label"
	case ExportEventNothing:
		return fmt.Sprintf("no event: %v", er.Reason)
	default:
		panic("unknown export result")
	}
}

func NewExportError(err error, id entity.Id) ExportResult {
	return ExportResult{
		ID:  id,
		Err: err,
	}
}

func NewExportNothing(id entity.Id, reason string) ExportResult {
	return ExportResult{
		ID:     id,
		Reason: reason,
		Event:  ExportEventNothing,
	}
}

func NewExportBug(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventBug,
	}
}

func NewExportComment(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventComment,
	}
}

func NewExportCommentEdition(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventCommentEdition,
	}
}

func NewExportStatusChange(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventStatusChange,
	}
}

func NewExportLabelChange(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventLabelChange,
	}
}

func NewExportTitleEdition(id entity.Id) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventTitleEdition,
	}
}
