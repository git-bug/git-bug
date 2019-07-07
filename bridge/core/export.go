package core

import "fmt"

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
	ID     string
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

func NewExportError(err error, reason string) ExportResult {
	return ExportResult{
		Err:    err,
		Reason: reason,
	}
}

func NewExportNothing(id string, reason string) ExportResult {
	return ExportResult{
		ID:     id,
		Reason: reason,
		Event:  ExportEventNothing,
	}
}

func NewExportBug(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventBug,
	}
}

func NewExportComment(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventComment,
	}
}

func NewExportCommentEdition(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventCommentEdition,
	}
}

func NewExportStatusChange(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventStatusChange,
	}
}

func NewExportLabelChange(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventLabelChange,
	}
}

func NewExportTitleEdition(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: ExportEventTitleEdition,
	}
}
