package entity

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/repository"
)

// func TestFoo(t *testing.T) {
// 	repo, err := repository.OpenGoGitRepo("~/dev/git-bug", nil)
// 	require.NoError(t, err)
//
// 	b, err := ReadBug(repo, Id("8b22e548c93a6ed23c31fd4e337c6286c3d1e5c9cae5537bc8e5842e11bd1099"))
// 	require.NoError(t, err)
//
// 	fmt.Println(b)
// }

type op1 struct {
	OperationType int    `json:"type"`
	Field1        string `json:"field_1"`
}

func newOp1(field1 string) *op1 {
	return &op1{OperationType: 1, Field1: field1}
}

func (o op1) Id() Id {
	data, _ := json.Marshal(o)
	return DeriveId(data)
}

func (o op1) Validate() error { return nil }

type op2 struct {
	OperationType int    `json:"type"`
	Field2        string `json:"field_2"`
}

func newOp2(field2 string) *op2 {
	return &op2{OperationType: 2, Field2: field2}
}

func (o op2) Id() Id {
	data, _ := json.Marshal(o)
	return DeriveId(data)
}

func (o op2) Validate() error { return nil }

var def = Definition{
	typename:             "foo",
	namespace:            "foos",
	operationUnmarshaler: unmarshaller,
	formatVersion:        1,
}

func unmarshaller(raw json.RawMessage) (Operation, error) {
	var t struct {
		OperationType int `json:"type"`
	}

	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	switch t.OperationType {
	case 1:
		op := &op1{}
		err := json.Unmarshal(raw, &op)
		return op, err
	case 2:
		op := &op2{}
		err := json.Unmarshal(raw, &op)
		return op, err
	default:
		return nil, fmt.Errorf("unknown operation type %v", t.OperationType)
	}
}

func TestWriteRead(t *testing.T) {
	repo := repository.NewMockRepo()

	entity := New(def)
	require.False(t, entity.NeedCommit())

	entity.Append(newOp1("foo"))
	entity.Append(newOp2("bar"))

	require.True(t, entity.NeedCommit())
	require.NoError(t, entity.CommitAdNeeded(repo))
	require.False(t, entity.NeedCommit())

	entity.Append(newOp2("foobar"))
	require.True(t, entity.NeedCommit())
	require.NoError(t, entity.CommitAdNeeded(repo))
	require.False(t, entity.NeedCommit())

	read, err := Read(def, repo, entity.Id())
	require.NoError(t, err)

	fmt.Println(*read)
}
