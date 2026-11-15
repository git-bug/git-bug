package board

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func TestApply(t *testing.T) {
	repo := repository.NewMockRepoClock()
	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	isaac, err := identity.NewIdentity(repo, "Isaac Newton", "isaac@newton.uk")
	require.NoError(t, err)

	unix := time.Now().Unix()

	bug := func(name string) Reference {
		return Reference{Namespace: "bugs", Id: entity.DeriveId([]byte(name))}
	}

	// newBoard returns the snapshot of a board with the columns Backlog, Doing and
	// Done, at a0, a1 and a2, with their ids. Operations are then applied to it in
	// the order they would be sorted in.
	newBoard := func() (snap *Snapshot, backlog, doing, done entity.Id) {
		snap = &Snapshot{}
		NewCreateOp(rene, unix, "Roadmap", "").Apply(snap)
		ids := make([]entity.Id, 3)
		for i, name := range []string{"Backlog", "Doing", "Done"} {
			op := NewAddColumnOp(rene, unix, name, fmt.Sprintf("a%d", i), i == 0)
			op.Apply(snap)
			ids[i] = op.Id()
		}
		return snap, ids[0], ids[1], ids[2]
	}

	// columns lists the columns of a snapshot, by name and position
	columns := func(snap *Snapshot) []string {
		var result []string
		for _, column := range snap.Columns {
			result = append(result, column.Name+" "+column.Position)
		}
		return result
	}

	// items lists the items of a snapshot: the title of a draft or the namespace of
	// an entity, the name of the column or - for none, and the position
	items := func(snap *Snapshot) []string {
		var result []string
		for _, item := range snap.Items {
			content := item.Title
			if !item.IsDraft() {
				content = item.Entity.Namespace
			}
			column := "-"
			if i, ok := snap.columnIndex(item.Column); ok {
				column = snap.Columns[i].Name
			} else {
				require.Empty(t, item.Column)
			}
			result = append(result, content+" "+column+" "+item.Position)
		}
		return result
	}

	t.Run("create", func(t *testing.T) {
		snap := &Snapshot{}
		create := NewCreateOp(rene, unix, "Roadmap", "What we work on")
		create.Apply(snap)

		require.Equal(t, create.Id(), snap.Id())
		require.Equal(t, "Roadmap", snap.Title)
		require.Equal(t, "What we work on", snap.Description)
		require.Empty(t, snap.Columns)
		require.Empty(t, snap.Items)

		// a second create is ignored
		NewCreateOp(rene, unix, "Other", "").Apply(snap)
		require.Equal(t, create.Id(), snap.Id())
		require.Equal(t, "Roadmap", snap.Title)

		NewSetTitleOp(rene, unix, "Roadmap 2021", "Roadmap").Apply(snap)
		NewSetDescriptionOp(rene, unix, "", "What we work on").Apply(snap)
		require.Equal(t, "Roadmap 2021", snap.Title)
		require.Empty(t, snap.Description)
	})

	t.Run("columns", func(t *testing.T) {
		snap, backlog, doing, _ := newBoard()
		require.Equal(t, []string{"Backlog a0", "Doing a1", "Done a2"}, columns(snap))
		require.True(t, snap.Columns[0].PriorityOrdered)
		require.False(t, snap.Columns[1].PriorityOrdered)
		require.Equal(t, entity.CombineIds(snap.Id(), backlog), snap.Columns[0].CombinedId)

		// each value is changed on its own
		name, position, ordered := "In review", "a3", true
		NewEditColumnOp(rene, unix, doing, &name, nil, nil).Apply(snap)
		NewEditColumnOp(rene, unix, doing, nil, &position, nil).Apply(snap)
		NewEditColumnOp(rene, unix, doing, nil, nil, &ordered).Apply(snap)
		require.Equal(t, []string{"Backlog a0", "Done a2", "In review a3"}, columns(snap))
		require.True(t, snap.Columns[2].PriorityOrdered)
		require.Equal(t, doing, snap.Columns[2].Id)

		// two columns may have the same name
		NewAddColumnOp(rene, unix, "Done", "a4", false).Apply(snap)
		require.Equal(t, []string{"Backlog a0", "Done a2", "In review a3", "Done a4"}, columns(snap))

		// a column removed doesn't come back
		NewRemoveColumnOp(rene, unix, backlog).Apply(snap)
		NewRemoveColumnOp(rene, unix, backlog).Apply(snap)
		name = "Later"
		NewEditColumnOp(rene, unix, backlog, &name, nil, nil).Apply(snap)
		require.Equal(t, []string{"Done a2", "In review a3", "Done a4"}, columns(snap))

		column, err := snap.SearchColumn(entity.CombineIds(snap.Id(), doing))
		require.NoError(t, err)
		require.Equal(t, "In review", column.Name)
		_, err = snap.SearchColumn(entity.CombineIds(snap.Id(), backlog))
		require.Error(t, err)
	})

	t.Run("items", func(t *testing.T) {
		snap, backlog, doing, _ := newBoard()

		draft := NewAddItemDraftOp(rene, unix, "Write the notes", backlog, "a1")
		draft.Apply(snap)
		added := NewAddItemEntityOp(isaac, unix, bug("parser"), backlog, "a0")
		added.Apply(snap)
		unplaced := NewAddItemDraftOp(rene, unix, "Try the parser", "", "a2")
		unplaced.Apply(snap)
		require.Equal(t, []string{"bugs Backlog a0", "Write the notes Backlog a1", "Try the parser - a2"}, items(snap))

		item := snap.Items[0]
		require.Equal(t, added.Id(), item.Id)
		require.Equal(t, entity.CombineIds(snap.Id(), added.Id()), item.CombinedId)
		require.Equal(t, isaac.Id(), item.Author.Id())
		require.Equal(t, bug("parser"), item.Entity)
		require.False(t, item.IsDraft())
		require.True(t, snap.Items[1].IsDraft())

		// within its column, then to another column, then out of any column
		NewMoveItemOp(rene, unix, draft.Id(), nil, "Zz").Apply(snap)
		require.Equal(t, []string{"Write the notes Backlog Zz", "bugs Backlog a0", "Try the parser - a2"}, items(snap))
		NewMoveItemOp(rene, unix, added.Id(), &doing, "a3").Apply(snap)
		require.Equal(t, []string{"Write the notes Backlog Zz", "Try the parser - a2", "bugs Doing a3"}, items(snap))
		NewMoveItemOp(rene, unix, draft.Id(), new(entity.Id), "a4").Apply(snap)
		require.Equal(t, []string{"Try the parser - a2", "bugs Doing a3", "Write the notes - a4"}, items(snap))

		require.Len(t, snap.ItemsOf(doing), 1)
		require.Len(t, snap.ItemsOf(""), 2)
		require.Empty(t, snap.ItemsOf(backlog))

		NewEditDraftOp(rene, unix, unplaced.Id(), "Try the parser on the tests").Apply(snap)
		require.Equal(t, "Try the parser on the tests", snap.Items[0].Title)

		// a converted draft keeps its id, its column and its position
		NewConvertDraftOp(rene, unix, unplaced.Id(), bug("tests")).Apply(snap)
		require.Equal(t, []string{"bugs - a2", "bugs Doing a3", "Write the notes - a4"}, items(snap))
		require.Equal(t, unplaced.Id(), snap.Items[0].Id)
		require.Equal(t, bug("tests"), snap.Items[0].Entity)
		require.Empty(t, snap.Items[0].Title)

		NewRemoveItemOp(rene, unix, added.Id()).Apply(snap)
		require.Equal(t, []string{"bugs - a2", "Write the notes - a4"}, items(snap))

		found, err := snap.SearchItem(entity.CombineIds(snap.Id(), draft.Id()))
		require.NoError(t, err)
		require.Equal(t, "Write the notes", found.Title)
		_, err = snap.SearchItem(entity.CombineIds(snap.Id(), added.Id()))
		require.Error(t, err)

		require.Len(t, snap.Actors, 2)
	})

	t.Run("order", func(t *testing.T) {
		snap, backlog, _, _ := newBoard()

		// items with the same position are listed by id, whatever the order they
		// were added in
		for i := range 5 {
			NewAddItemDraftOp(rene, unix, fmt.Sprintf("draft %d", i), backlog, "a0V").Apply(snap)
		}
		NewAddItemDraftOp(rene, unix, "first", backlog, "a0").Apply(snap)
		NewAddItemDraftOp(rene, unix, "last", "", "a1").Apply(snap)

		require.Equal(t, "first", snap.Items[0].Title)
		require.Equal(t, "last", snap.Items[6].Title)
		for i := 2; i < 6; i++ {
			require.Equal(t, "a0V", snap.Items[i].Position)
			require.Less(t, snap.Items[i-1].Id, snap.Items[i].Id)
		}

		// and so are columns
		snap = &Snapshot{}
		NewCreateOp(rene, unix, "Roadmap", "").Apply(snap)
		for i := range 5 {
			NewAddColumnOp(rene, unix, fmt.Sprintf("column %d", i), "a0", false).Apply(snap)
		}
		for i := 1; i < 5; i++ {
			require.Less(t, snap.Columns[i-1].Id, snap.Columns[i].Id)
		}
	})

	t.Run("missing targets", func(t *testing.T) {
		t.Run("item moved to a removed column", func(t *testing.T) {
			snap, backlog, doing, _ := newBoard()
			x := NewAddItemDraftOp(rene, unix, "X", backlog, "a0")
			x.Apply(snap)
			NewRemoveColumnOp(rene, unix, doing).Apply(snap)
			NewMoveItemOp(rene, unix, x.Id(), &doing, "a3").Apply(snap)
			require.Equal(t, []string{"X - a3"}, items(snap))
		})

		t.Run("column removed after an item was moved to it", func(t *testing.T) {
			snap, backlog, doing, _ := newBoard()
			x := NewAddItemDraftOp(rene, unix, "X", backlog, "a0")
			x.Apply(snap)
			NewMoveItemOp(rene, unix, x.Id(), &doing, "a3").Apply(snap)
			NewRemoveColumnOp(rene, unix, doing).Apply(snap)
			require.Equal(t, []string{"X - a3"}, items(snap))
		})

		t.Run("item added to a column that is not present", func(t *testing.T) {
			snap, _, doing, _ := newBoard()
			NewRemoveColumnOp(rene, unix, doing).Apply(snap)
			NewAddItemDraftOp(rene, unix, "X", doing, "a0").Apply(snap)
			NewAddItemEntityOp(rene, unix, bug("parser"), entity.DeriveId([]byte("never added")), "a1").Apply(snap)
			require.Equal(t, []string{"X - a0", "bugs - a1"}, items(snap))
		})

		t.Run("removed item", func(t *testing.T) {
			snap, backlog, _, done := newBoard()
			x := NewAddItemDraftOp(rene, unix, "X", backlog, "a0")
			x.Apply(snap)
			NewRemoveItemOp(rene, unix, x.Id()).Apply(snap)
			NewMoveItemOp(rene, unix, x.Id(), &done, "a3").Apply(snap)
			NewEditDraftOp(rene, unix, x.Id(), "Y").Apply(snap)
			NewConvertDraftOp(rene, unix, x.Id(), bug("parser")).Apply(snap)
			NewRemoveItemOp(rene, unix, x.Id()).Apply(snap)
			require.Empty(t, snap.Items)
		})

		t.Run("entity added twice", func(t *testing.T) {
			snap, backlog, doing, done := newBoard()
			parser := bug("parser")
			x := NewAddItemEntityOp(rene, unix, parser, backlog, "a0")
			x.Apply(snap)
			y := NewAddItemEntityOp(rene, unix, parser, done, "a5")
			y.Apply(snap)
			require.Equal(t, []string{"bugs Backlog a0"}, items(snap))
			require.Equal(t, x.Id(), snap.Items[0].Id)

			// what follows the second operation designates the bug, and applies to the
			// item of the first
			NewMoveItemOp(rene, unix, parser.Id, &doing, "a6").Apply(snap)
			require.Equal(t, []string{"bugs Doing a6"}, items(snap))
			// no item has the id of the second operation
			NewMoveItemOp(rene, unix, y.Id(), &done, "a7").Apply(snap)
			require.Equal(t, []string{"bugs Doing a6"}, items(snap))

			// under another namespace, the id is still the one of an entity that the
			// board holds
			NewAddItemEntityOp(rene, unix, Reference{Namespace: "other", Id: parser.Id}, done, "a8").Apply(snap)
			require.Equal(t, []string{"bugs Doing a6"}, items(snap))

			// another entity is another item, and so is the same one once its item is
			// removed: the id of the entity then designates the new item
			NewAddItemEntityOp(rene, unix, bug("lexer"), backlog, "a1").Apply(snap)
			NewRemoveItemOp(rene, unix, parser.Id).Apply(snap)
			z := NewAddItemEntityOp(rene, unix, parser, done, "a2")
			z.Apply(snap)
			require.Equal(t, []string{"bugs Backlog a1", "bugs Done a2"}, items(snap))
			require.Equal(t, z.Id(), snap.Items[1].Id)
			NewMoveItemOp(rene, unix, parser.Id, nil, "a3").Apply(snap)
			require.Equal(t, []string{"bugs Backlog a1", "bugs Done a3"}, items(snap))
			require.Equal(t, z.Id(), snap.Items[1].Id)
		})

		t.Run("move within a column and move to another column", func(t *testing.T) {
			snap, backlog, _, done := newBoard()
			x := NewAddItemDraftOp(rene, unix, "X", backlog, "a0")
			x.Apply(snap)
			NewMoveItemOp(rene, unix, x.Id(), &done, "a4").Apply(snap)
			NewMoveItemOp(rene, unix, x.Id(), nil, "a1").Apply(snap)
			require.Equal(t, []string{"X Done a1"}, items(snap))
		})

		t.Run("converted draft", func(t *testing.T) {
			snap, backlog, doing, _ := newBoard()
			x := NewAddItemDraftOp(rene, unix, "X", backlog, "a0")
			x.Apply(snap)
			NewConvertDraftOp(rene, unix, x.Id(), bug("parser")).Apply(snap)
			NewEditDraftOp(rene, unix, x.Id(), "Y").Apply(snap)
			NewConvertDraftOp(rene, unix, x.Id(), bug("lexer")).Apply(snap)
			require.Equal(t, []string{"bugs Backlog a0"}, items(snap))
			require.Equal(t, bug("parser"), snap.Items[0].Entity)
			require.Empty(t, snap.Items[0].Title)

			// its id still designates it, for a move made on the draft, and so does the
			// id of the bug
			NewMoveItemOp(rene, unix, x.Id(), &doing, "a3").Apply(snap)
			require.Equal(t, []string{"bugs Doing a3"}, items(snap))
			NewMoveItemOp(rene, unix, bug("parser").Id, nil, "a4").Apply(snap)
			require.Equal(t, []string{"bugs Doing a4"}, items(snap))
			require.Equal(t, x.Id(), snap.Items[0].Id)
		})

		t.Run("draft converted into an entity that the board holds", func(t *testing.T) {
			snap, backlog, _, _ := newBoard()
			NewAddItemEntityOp(rene, unix, bug("parser"), backlog, "a0").Apply(snap)
			x := NewAddItemDraftOp(rene, unix, "X", backlog, "a1")
			x.Apply(snap)
			NewConvertDraftOp(rene, unix, x.Id(), bug("parser")).Apply(snap)
			require.Equal(t, []string{"bugs Backlog a0", "X Backlog a1"}, items(snap))
		})
	})

	t.Run("clone", func(t *testing.T) {
		snap, backlog, doing, _ := newBoard()
		x := NewAddItemDraftOp(rene, unix, "X", backlog, "a0")
		x.Apply(snap)

		clone := snap.Clone()
		NewMoveItemOp(isaac, unix, x.Id(), &doing, "a1").Apply(clone)
		NewEditDraftOp(isaac, unix, x.Id(), "Y").Apply(clone)
		NewAddItemDraftOp(isaac, unix, "Z", backlog, "Zz").Apply(clone)
		NewRemoveColumnOp(isaac, unix, backlog).Apply(clone)
		name := "In review"
		NewEditColumnOp(isaac, unix, doing, &name, nil, nil).Apply(clone)

		require.Equal(t, []string{"Backlog a0", "Doing a1", "Done a2"}, columns(snap))
		require.Equal(t, []string{"X Backlog a0"}, items(snap))
		require.Len(t, snap.Actors, 1)

		require.Equal(t, []string{"In review a1", "Done a2"}, columns(clone))
		require.Equal(t, []string{"Z - Zz", "Y In review a1"}, items(clone))
		require.Len(t, clone.Actors, 2)
	})
}
