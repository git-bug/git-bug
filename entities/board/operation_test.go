package board

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
	"github.com/git-bug/git-bug/repository"
)

func TestOperations(t *testing.T) {
	repo := repository.NewMockRepoClock()
	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	unix := time.Now().Unix()

	target := entity.DeriveId([]byte("target"))
	column := entity.DeriveId([]byte("column"))
	bug := Reference{Namespace: "bugs", Id: entity.DeriveId([]byte("bug"))}
	name, position, ordered, notOrdered := "Doing", "a0V", true, false
	empty, invalid := "", entity.Id("invalid")

	// makers build one operation of each type and form, for any author and time
	makers := []func(author identity.Interface, unixTime int64) Operation{
		func(a identity.Interface, u int64) Operation { return NewCreateOp(a, u, "title", "description") },
		func(a identity.Interface, u int64) Operation { return NewCreateOp(a, u, "title", "") },
		func(a identity.Interface, u int64) Operation { return NewSetTitleOp(a, u, "title", "was") },
		func(a identity.Interface, u int64) Operation { return NewSetDescriptionOp(a, u, "description", "was") },
		func(a identity.Interface, u int64) Operation { return NewSetDescriptionOp(a, u, "", "was") },
		func(a identity.Interface, u int64) Operation { return NewAddColumnOp(a, u, "Backlog", "a0", true) },
		func(a identity.Interface, u int64) Operation { return NewAddColumnOp(a, u, "Done", "a1", false) },
		func(a identity.Interface, u int64) Operation { return NewEditColumnOp(a, u, target, &name, nil, nil) },
		func(a identity.Interface, u int64) Operation {
			return NewEditColumnOp(a, u, target, nil, &position, nil)
		},
		func(a identity.Interface, u int64) Operation {
			return NewEditColumnOp(a, u, target, nil, nil, &notOrdered)
		},
		func(a identity.Interface, u int64) Operation {
			return NewEditColumnOp(a, u, target, &name, &position, &ordered)
		},
		func(a identity.Interface, u int64) Operation { return NewRemoveColumnOp(a, u, target) },
		func(a identity.Interface, u int64) Operation { return NewAddItemDraftOp(a, u, "title", column, "a0") },
		func(a identity.Interface, u int64) Operation { return NewAddItemDraftOp(a, u, "title", "", "a0") },
		func(a identity.Interface, u int64) Operation { return NewAddItemEntityOp(a, u, bug, column, "a0") },
		func(a identity.Interface, u int64) Operation { return NewAddItemEntityOp(a, u, bug, "", "a0") },
		func(a identity.Interface, u int64) Operation { return NewMoveItemOp(a, u, target, nil, "a0") },
		func(a identity.Interface, u int64) Operation {
			return NewMoveItemOp(a, u, target, new(entity.Id), "a0")
		},
		func(a identity.Interface, u int64) Operation { return NewMoveItemOp(a, u, target, &column, "a0") },
		func(a identity.Interface, u int64) Operation { return NewRemoveItemOp(a, u, target) },
		func(a identity.Interface, u int64) Operation { return NewEditDraftOp(a, u, target, "title") },
		func(a identity.Interface, u int64) Operation { return NewConvertDraftOp(a, u, target, bug) },
		func(a identity.Interface, u int64) Operation {
			return NewSetMetadataOp(a, u, target, map[string]string{"key": "value"})
		},
	}

	t.Run("validate", func(t *testing.T) {
		for i, maker := range makers {
			require.NoError(t, maker(rene, unix).Validate(), i)
		}

		bad := []Operation{
			// opbase
			&CreateOperation{OpBase: dag.NewOpBase(CreateOp, rene, 0), Title: "title"},
			&CreateOperation{OpBase: dag.NewOpBase(SetTitleOp, rene, unix), Title: "title"},

			NewCreateOp(rene, unix, "", ""),
			NewCreateOp(rene, unix, "multi\nline", ""),
			NewCreateOp(rene, unix, "title\u001b", ""),
			NewCreateOp(rene, unix, "title", "multi\nline"),
			NewSetTitleOp(rene, unix, "", "was"),
			NewSetTitleOp(rene, unix, "multi\nline", "was"),
			NewSetTitleOp(rene, unix, "title", "multi\nline"),
			NewSetDescriptionOp(rene, unix, "multi\nline", "was"),
			NewSetDescriptionOp(rene, unix, "description", "multi\nline"),

			NewAddColumnOp(rene, unix, "", "a0", false),
			NewAddColumnOp(rene, unix, "multi\nline", "a0", false),
			NewAddColumnOp(rene, unix, "Backlog", "", false),
			NewAddColumnOp(rene, unix, "Backlog", "a00", false),
			NewEditColumnOp(rene, unix, target, nil, nil, nil),
			NewEditColumnOp(rene, unix, invalid, &name, nil, nil),
			NewEditColumnOp(rene, unix, "", &name, nil, nil),
			NewEditColumnOp(rene, unix, target, &empty, nil, nil),
			NewEditColumnOp(rene, unix, target, nil, &empty, nil),
			NewRemoveColumnOp(rene, unix, invalid),
			NewRemoveColumnOp(rene, unix, ""),

			NewAddItemDraftOp(rene, unix, "", column, "a0"),
			NewAddItemDraftOp(rene, unix, "multi\nline", column, "a0"),
			NewAddItemDraftOp(rene, unix, "title", invalid, "a0"),
			NewAddItemDraftOp(rene, unix, "title", column, "a"),
			NewAddItemEntityOp(rene, unix, Reference{}, column, "a0"),
			NewAddItemEntityOp(rene, unix, Reference{Namespace: "bugs"}, column, "a0"),
			NewAddItemEntityOp(rene, unix, Reference{Namespace: "bugs", Id: invalid}, column, "a0"),
			NewAddItemEntityOp(rene, unix, Reference{Id: bug.Id}, column, "a0"),
			NewAddItemEntityOp(rene, unix, Reference{Namespace: "multi\nline", Id: bug.Id}, column, "a0"),
			NewAddItemEntityOp(rene, unix, bug, invalid, "a0"),
			NewAddItemEntityOp(rene, unix, bug, column, ""),
			NewMoveItemOp(rene, unix, invalid, nil, "a0"),
			NewMoveItemOp(rene, unix, "", nil, "a0"),
			NewMoveItemOp(rene, unix, target, &invalid, "a0"),
			NewMoveItemOp(rene, unix, target, nil, ""),
			NewRemoveItemOp(rene, unix, invalid),
			NewEditDraftOp(rene, unix, invalid, "title"),
			NewEditDraftOp(rene, unix, target, ""),
			NewEditDraftOp(rene, unix, target, "multi\nline"),
			NewConvertDraftOp(rene, unix, invalid, bug),
			NewConvertDraftOp(rene, unix, target, Reference{}),
			NewConvertDraftOp(rene, unix, target, Reference{Namespace: "bugs", Id: invalid}),
		}
		for i, op := range bad {
			require.Error(t, op.Validate(), i)
		}
	})

	t.Run("serialize", func(t *testing.T) {
		for _, maker := range makers {
			dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (Operation, entity.Resolvers) {
				return maker(author, unixTime), nil
			})
		}
	})

	t.Run("decode", func(t *testing.T) {
		// decode reads the JSON of an operation, made of the common fields and of
		// the given members
		decode := func(members string) (dag.Operation, error) {
			raw := `{"timestamp":1609459200,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAA",` + members + `}`
			return operationUnmarshaler(t.Context(), json.RawMessage(raw), nil)
		}
		id := `"` + target.String() + `"`

		// the column of a move: absent, null or an id
		op, err := decode(`"type":9,"item":` + id + `,"position":"a0"`)
		require.NoError(t, err)
		require.Nil(t, op.(*MoveItemOperation).Column)

		op, err = decode(`"type":9,"item":` + id + `,"column":null,"position":"a0"`)
		require.NoError(t, err)
		require.NotNil(t, op.(*MoveItemOperation).Column)
		require.Empty(t, *op.(*MoveItemOperation).Column)

		op, err = decode(`"type":9,"item":` + id + `,"column":` + id + `,"position":"a0"`)
		require.NoError(t, err)
		require.Equal(t, target, *op.(*MoveItemOperation).Column)
		require.Equal(t, target, op.(*MoveItemOperation).Item)
		require.Equal(t, "a0", op.(*MoveItemOperation).Position)

		_, err = decode(`"type":9,"item":` + id + `,"column":"","position":"a0"`)
		require.Error(t, err)
		_, err = decode(`"type":9,"item":` + id + `,"column":12,"position":"a0"`)
		require.Error(t, err)

		// a member that may be absent is not null instead
		for _, members := range []string{
			`"type":4,"name":"Backlog","position":"a0","priority_ordered":null`,
			`"type":5,"column":` + id + `,"name":null`,
			`"type":5,"column":` + id + `,"name":"Doing","position":null`,
			`"type":5,"column":` + id + `,"name":"Doing","priority_ordered":null`,
			`"type":7,"title":"title","column":null,"position":"a0"`,
			`"type":8,"namespace":"bugs","entity":` + id + `,"column":null,"position":"a0"`,
		} {
			_, err := decode(members)
			require.Error(t, err, members)
		}

		// an absent member leaves the value of the column as it is
		op, err = decode(`"type":5,"column":` + id + `,"priority_ordered":false`)
		require.NoError(t, err)
		edit := op.(*EditColumnOperation)
		require.Nil(t, edit.Name)
		require.Nil(t, edit.Position)
		require.Equal(t, &notOrdered, edit.PriorityOrdered)

		// a member that this client doesn't know is ignored
		op, err = decode(`"type":4,"name":"Backlog","position":"a0","color":"#ff0000"`)
		require.NoError(t, err)
		require.Equal(t, "Backlog", op.(*AddColumnOperation).Name)
		require.False(t, op.(*AddColumnOperation).PriorityOrdered)

		// and so is an operation of a type that it doesn't know
		op, err = decode(`"type":1000,"whatever":true`)
		require.NoError(t, err)
		require.IsType(t, &dag.UnknownOperation[*Snapshot]{}, op)
	})
}
