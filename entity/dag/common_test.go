package dag

import (
	"encoding/json"
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

// This file contains an example dummy entity to be used in the tests

/*
 Operations
*/

type op1 struct {
	author identity.Interface

	OperationType int    `json:"type"`
	Field1        string `json:"field_1"`
}

func newOp1(author identity.Interface, field1 string) *op1 {
	return &op1{author: author, OperationType: 1, Field1: field1}
}

func (o op1) Id() entity.Id {
	data, _ := json.Marshal(o)
	return entity.DeriveId(data)
}

func (o op1) Author() identity.Interface {
	return o.author
}

func (o op1) Validate() error { return nil }

type op2 struct {
	author identity.Interface

	OperationType int    `json:"type"`
	Field2        string `json:"field_2"`
}

func newOp2(author identity.Interface, field2 string) *op2 {
	return &op2{author: author, OperationType: 2, Field2: field2}
}

func (o op2) Id() entity.Id {
	data, _ := json.Marshal(o)
	return entity.DeriveId(data)
}

func (o op2) Author() identity.Interface {
	return o.author
}

func (o op2) Validate() error { return nil }

func unmarshaler(author identity.Interface, raw json.RawMessage) (Operation, error) {
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
		op.author = author
		return op, err
	case 2:
		op := &op2{}
		err := json.Unmarshal(raw, &op)
		op.author = author
		return op, err
	default:
		return nil, fmt.Errorf("unknown operation type %v", t.OperationType)
	}
}

/*
  Identities + repo + definition
*/

func makeTestContext() (repository.ClockedRepo, identity.Interface, identity.Interface, Definition) {
	repo := repository.NewMockRepo()

	id1, err := identity.NewIdentity(repo, "name1", "email1")
	if err != nil {
		panic(err)
	}
	err = id1.Commit(repo)
	if err != nil {
		panic(err)
	}
	id2, err := identity.NewIdentity(repo, "name2", "email2")
	if err != nil {
		panic(err)
	}
	err = id2.Commit(repo)
	if err != nil {
		panic(err)
	}

	resolver := identityResolverFunc(func(id entity.Id) (identity.Interface, error) {
		switch id {
		case id1.Id():
			return id1, nil
		case id2.Id():
			return id2, nil
		default:
			return nil, identity.ErrIdentityNotExist
		}
	})

	def := Definition{
		typename:             "foo",
		namespace:            "foos",
		operationUnmarshaler: unmarshaler,
		identityResolver:     resolver,
		formatVersion:        1,
	}

	return repo, id1, id2, def
}

type identityResolverFunc func(id entity.Id) (identity.Interface, error)

func (fn identityResolverFunc) ResolveIdentity(id entity.Id) (identity.Interface, error) {
	return fn(id)
}
