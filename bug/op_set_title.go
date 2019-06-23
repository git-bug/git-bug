package bug

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/timestamp"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &SetTitleOperation{}

// SetTitleOperation will change the title of a bug
type SetTitleOperation struct {
	OpBase
	Title string
	Was   string
}

func (op *SetTitleOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetTitleOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

func (op *SetTitleOperation) Apply(snapshot *Snapshot) {
	snapshot.Title = op.Title
	snapshot.addActor(op.Author)

	hash, err := op.Hash()
	if err != nil {
		// Should never error unless a programming error happened
		// (covered in OpBase.Validate())
		panic(err)
	}

	item := &SetTitleTimelineItem{
		hash:     hash,
		Author:   op.Author,
		UnixTime: timestamp.Timestamp(op.UnixTime),
		Title:    op.Title,
		Was:      op.Was,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetTitleOperation) Validate() error {
	if err := opBaseValidate(op, SetTitleOp); err != nil {
		return err
	}

	if text.Empty(op.Title) {
		return fmt.Errorf("title is empty")
	}

	if strings.Contains(op.Title, "\n") {
		return fmt.Errorf("title should be a single line")
	}

	if !text.Safe(op.Title) {
		return fmt.Errorf("title should be fully printable")
	}

	if strings.Contains(op.Was, "\n") {
		return fmt.Errorf("previous title should be a single line")
	}

	if !text.Safe(op.Was) {
		return fmt.Errorf("previous title should be fully printable")
	}

	return nil
}

// Workaround to avoid the inner OpBase.MarshalJSON overriding the outer op
// MarshalJSON
func (op *SetTitleOperation) MarshalJSON() ([]byte, error) {
	base, err := json.Marshal(op.OpBase)
	if err != nil {
		return nil, err
	}

	// revert back to a flat map to be able to add our own fields
	var data map[string]interface{}
	if err := json.Unmarshal(base, &data); err != nil {
		return nil, err
	}

	data["title"] = op.Title
	data["was"] = op.Was

	return json.Marshal(data)
}

// Workaround to avoid the inner OpBase.MarshalJSON overriding the outer op
// MarshalJSON
func (op *SetTitleOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Title string `json:"title"`
		Was   string `json:"was"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Title = aux.Title
	op.Was = aux.Was

	return nil
}

// Sign post method for gqlgen
func (op *SetTitleOperation) IsAuthored() {}

func NewSetTitleOp(author identity.Interface, unixTime int64, title string, was string) *SetTitleOperation {
	return &SetTitleOperation{
		OpBase: newOpBase(SetTitleOp, author, unixTime),
		Title:  title,
		Was:    was,
	}
}

type SetTitleTimelineItem struct {
	hash     git.Hash
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Title    string
	Was      string
}

func (s SetTitleTimelineItem) Hash() git.Hash {
	return s.hash
}

// Sign post method for gqlgen
func (s *SetTitleTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func SetTitle(b Interface, author identity.Interface, unixTime int64, title string) (*SetTitleOperation, error) {
	it := NewOperationIterator(b)

	var lastTitleOp Operation
	for it.Next() {
		op := it.Value()
		if op.base().OperationType == SetTitleOp {
			lastTitleOp = op
		}
	}

	var was string
	if lastTitleOp != nil {
		was = lastTitleOp.(*SetTitleOperation).Title
	} else {
		was = b.FirstOp().(*CreateOperation).Title
	}

	setTitleOp := NewSetTitleOp(author, unixTime, title, was)

	if err := setTitleOp.Validate(); err != nil {
		return nil, err
	}

	b.Append(setTitleOp)
	return setTitleOp, nil
}
