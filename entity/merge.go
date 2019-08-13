package entity

import (
	"fmt"
)

// MergeStatus represent the result of a merge operation of an entity
type MergeStatus int

const (
	_ MergeStatus = iota
	MergeStatusNew
	MergeStatusInvalid
	MergeStatusUpdated
	MergeStatusNothing
	MergeStatusError
)

type MergeResult struct {
	// Err is set when a terminal error occur in the process
	Err error

	Id     Id
	Status MergeStatus

	// Only set for invalid status
	Reason string

	// Not set for invalid status
	Entity Interface
}

func (mr MergeResult) String() string {
	switch mr.Status {
	case MergeStatusNew:
		return "new"
	case MergeStatusInvalid:
		return fmt.Sprintf("invalid data: %s", mr.Reason)
	case MergeStatusUpdated:
		return "updated"
	case MergeStatusNothing:
		return "nothing to do"
	case MergeStatusError:
		return fmt.Sprintf("merge error on %s: %s", mr.Id, mr.Err.Error())
	default:
		panic("unknown merge status")
	}
}

func NewMergeError(err error, id Id) MergeResult {
	return MergeResult{
		Err:    err,
		Id:     id,
		Status: MergeStatusError,
	}
}

func NewMergeStatus(status MergeStatus, id Id, entity Interface) MergeResult {
	return MergeResult{
		Id:     id,
		Status: status,

		// Entity is not set for an invalid merge result
		Entity: entity,
	}
}

func NewMergeInvalidStatus(id Id, reason string) MergeResult {
	return MergeResult{
		Id:     id,
		Status: MergeStatusInvalid,
		Reason: reason,
	}
}
