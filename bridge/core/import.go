package core

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
)

type ImportEvent int

const (
	_ ImportEvent = iota
	ImportEventBug
	ImportEventComment
	ImportEventCommentEdition
	ImportEventStatusChange
	ImportEventTitleEdition
	ImportEventLabelChange
	ImportEventIdentity
	ImportEventNothing
	ImportEventError
)

// ImportResult is an event that is emitted during the import process, to
// allow calling code to report on what is happening, collect metrics or
// display meaningful errors if something went wrong.
type ImportResult struct {
	Err    error
	Event  ImportEvent
	ID     entity.Id
	Reason string
}

func (er ImportResult) String() string {
	switch er.Event {
	case ImportEventBug:
		return fmt.Sprintf("new issue: %s", er.ID)
	case ImportEventComment:
		return fmt.Sprintf("new comment: %s", er.ID)
	case ImportEventCommentEdition:
		return fmt.Sprintf("updated comment: %s", er.ID)
	case ImportEventStatusChange:
		return fmt.Sprintf("changed status: %s", er.ID)
	case ImportEventTitleEdition:
		return fmt.Sprintf("changed title: %s", er.ID)
	case ImportEventLabelChange:
		return fmt.Sprintf("changed label: %s", er.ID)
	case ImportEventIdentity:
		return fmt.Sprintf("new identity: %s", er.ID)
	case ImportEventNothing:
		if er.ID != "" {
			return fmt.Sprintf("ignoring import event %s: %s", er.ID, er.Reason)
		}
		return fmt.Sprintf("ignoring event: %s", er.Reason)
	case ImportEventError:
		if er.ID != "" {
			return fmt.Sprintf("import error at id %s: %s", er.ID, er.Err.Error())
		}
		return fmt.Sprintf("import error: %s", er.Err.Error())
	default:
		panic("unknown import result")
	}
}

func NewImportError(err error, id entity.Id) ImportResult {
	return ImportResult{
		Err:   err,
		ID:    id,
		Event: ImportEventError,
	}
}

func NewImportNothing(id entity.Id, reason string) ImportResult {
	return ImportResult{
		ID:     id,
		Reason: reason,
		Event:  ImportEventNothing,
	}
}

func NewImportBug(id entity.Id) ImportResult {
	return ImportResult{
		ID:    id,
		Event: ImportEventBug,
	}
}

func NewImportComment(id entity.Id) ImportResult {
	return ImportResult{
		ID:    id,
		Event: ImportEventComment,
	}
}

func NewImportCommentEdition(id entity.Id) ImportResult {
	return ImportResult{
		ID:    id,
		Event: ImportEventCommentEdition,
	}
}

func NewImportStatusChange(id entity.Id) ImportResult {
	return ImportResult{
		ID:    id,
		Event: ImportEventStatusChange,
	}
}

func NewImportLabelChange(id entity.Id) ImportResult {
	return ImportResult{
		ID:    id,
		Event: ImportEventLabelChange,
	}
}

func NewImportTitleEdition(id entity.Id) ImportResult {
	return ImportResult{
		ID:    id,
		Event: ImportEventTitleEdition,
	}
}

func NewImportIdentity(id entity.Id) ImportResult {
	return ImportResult{
		ID:    id,
		Event: ImportEventIdentity,
	}
}
