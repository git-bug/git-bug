package core

import "fmt"

type EventStatus int

const (
	_ EventStatus = iota
	EventStatusBug
	EventStatusComment
	EventStatusCommentEdition
	EventStatusStatusChange
	EventStatusTitleEdition
	EventStatusLabelChange
	EventStatusNothing
)

type ExportResult struct {
	Err    error
	Event  EventStatus
	ID     string
	Reason string
}

func (er ExportResult) String() string {
	switch er.Event {
	case EventStatusBug:
		return "new issue"
	case EventStatusComment:
		return "new comment"
	case EventStatusCommentEdition:
		return "updated comment"
	case EventStatusStatusChange:
		return "changed status"
	case EventStatusTitleEdition:
		return "changed title"
	case EventStatusLabelChange:
		return "changed label"
	case EventStatusNothing:
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
		Event:  EventStatusNothing,
	}
}

func NewExportBug(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: EventStatusBug,
	}
}

func NewExportComment(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: EventStatusComment,
	}
}

func NewExportCommentEdition(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: EventStatusCommentEdition,
	}
}

func NewExportStatusChange(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: EventStatusStatusChange,
	}
}

func NewExportLabelChange(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: EventStatusLabelChange,
	}
}

func NewExportTitleEdition(id string) ExportResult {
	return ExportResult{
		ID:    id,
		Event: EventStatusTitleEdition,
	}
}
