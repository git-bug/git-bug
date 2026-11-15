package board

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

func TestBoard(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	require.NoError(t, rene.Commit(t.Context(), repo))

	unix := time.Now().Unix()

	t.Run("validate", func(t *testing.T) {
		// created with its columns, in the same pack
		b, _, err := Create(rene, unix, "Roadmap", "", nil)
		require.NoError(t, err)
		_, _, err = AddColumn(b, rene, unix, "Backlog", "a0", true, nil)
		require.NoError(t, err)
		require.NoError(t, b.Validate())

		// no Create first
		b = NewBoard()
		_, _, err = AddColumn(b, rene, unix, "Backlog", "a0", true, nil)
		require.NoError(t, err)
		require.Error(t, b.Validate())

		// a second Create
		b, _, err = Create(rene, unix, "Roadmap", "", nil)
		require.NoError(t, err)
		b.Append(NewCreateOp(rene, unix, "Roadmap", ""))
		require.Error(t, b.Validate())

		// an operation that is not valid is not staged
		b, _, err = Create(rene, unix, "Roadmap", "", nil)
		require.NoError(t, err)
		_, _, err = AddColumn(b, rene, unix, "Backlog", "not a position", true, nil)
		require.Error(t, err)
		require.Len(t, b.Operations(), 1)
	})

	// The test vectors of the spec. testdata/board.json is a copy of the file of
	// the same name in https://github.com/git-bug/spec. Its packs are written in
	// git as the spec lays them out, and the board read from them must hold the
	// same operations and give the same snapshots.
	//
	// The apply_examples of the file are abbreviated, and not read here: TestApply
	// has the same cases.
	t.Run("spec vectors", func(t *testing.T) {
		ctx := t.Context()

		type column struct {
			Id              entity.Id `json:"id"`
			Name            string    `json:"name"`
			Position        string    `json:"position"`
			PriorityOrdered bool      `json:"priority_ordered"`
		}
		type item struct {
			Id entity.Id `json:"id"`
			// Content is the title of a draft, or the reference to an entity
			Content struct {
				Draft     string    `json:"draft"`
				Namespace string    `json:"namespace"`
				Entity    entity.Id `json:"entity"`
			} `json:"content"`
			// Column is the name of the column of the item, nil for none
			Column   *string `json:"column"`
			Position string  `json:"position"`
		}
		// placed is an item of the order example: an id at a position
		type placed struct {
			Id       entity.Id `json:"id"`
			Position string    `json:"position"`
		}
		var vectors struct {
			Packs []struct {
				OpsBlob struct {
					Ops []json.RawMessage `json:"ops"`
				} `json:"ops_blob"`
				// Ids are the ids of operations of the pack, under a description of
				// what each identifies
				Ids map[string]entity.Id `json:"ids"`
			} `json:"operation_pack_examples"`
			Snapshots []struct {
				// AfterPacks is the number of packs that give the snapshot
				AfterPacks  int      `json:"after_packs"`
				Title       string   `json:"title"`
				Description string   `json:"description_field"`
				Columns     []column `json:"columns"`
				Items       []item   `json:"items"`
			} `json:"snapshot_examples"`
			Order struct {
				Items []placed    `json:"items"`
				Order []entity.Id `json:"order"`
			} `json:"order_example"`
			IdDerivation []struct {
				Raw        string    `json:"raw"`
				ExpectedId entity.Id `json:"expected_id"`
			} `json:"id_derivation"`
		}
		data, err := os.ReadFile(filepath.Join("testdata", "board.json"))
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &vectors))
		require.NotEmpty(t, vectors.Packs)
		require.NotEmpty(t, vectors.Snapshots)
		require.NotEmpty(t, vectors.Order.Items)
		require.NotEmpty(t, vectors.IdDerivation)

		// columnsOf and itemsOf give the columns and the items of a snapshot the way
		// the vectors write them
		columnsOf := func(snap *Snapshot) []column {
			result := make([]column, 0, len(snap.Columns))
			for _, c := range snap.Columns {
				result = append(result, column{Id: c.Id, Name: c.Name, Position: c.Position, PriorityOrdered: c.PriorityOrdered})
			}
			return result
		}
		itemsOf := func(snap *Snapshot) []item {
			result := make([]item, 0, len(snap.Items))
			for _, i := range snap.Items {
				v := item{Id: i.Id, Position: i.Position}
				if i.IsDraft() {
					v.Content.Draft = i.Title
				} else {
					v.Content.Namespace = i.Entity.Namespace
					v.Content.Entity = i.Entity.Id
				}
				if at, ok := snap.columnIndex(i.Column); ok {
					v.Column = &snap.Columns[at].Name
				}
				result = append(result, v)
			}
			return result
		}

		// raws are the operations of the vectors as git holds them: compact JSON
		var raws []string
		// packOf gives the index of the pack of each operation, by id
		packOf := map[entity.Id]int{}
		for i, pack := range vectors.Packs {
			for _, op := range pack.OpsBlob.Ops {
				var raw bytes.Buffer
				require.NoError(t, json.Compact(&raw, op))
				raws = append(raws, raw.String())
				packOf[entity.DeriveId(raw.Bytes())] = i
			}
		}
		boardId := entity.DeriveId([]byte(raws[0]))

		// Each pack is a commit on top of the previous one. Its tree holds the
		// operations, the format version, and the clocks, counted from 1. The author
		// is not part of the JSON of an operation, and is the one of this test.
		emptyBlob, err := repo.StoreData(ctx, nil)
		require.NoError(t, err)
		var head repository.Hash
		written := 0
		for i, pack := range vectors.Packs {
			ops := raws[written : written+len(pack.OpsBlob.Ops)]
			written += len(ops)

			blob, err := repo.StoreData(ctx, fmt.Appendf(nil, `{"author":{"id":%q},"ops":[%s]}`, rene.Id(), strings.Join(ops, ",")))
			require.NoError(t, err)
			entries := []repository.TreeEntry{
				{ObjectType: repository.Blob, Hash: blob, Name: "ops"},
				{ObjectType: repository.Blob, Hash: emptyBlob, Name: "version-1"},
				{ObjectType: repository.Blob, Hash: emptyBlob, Name: fmt.Sprintf("edit-clock-%d", i+1)},
			}
			var parents []repository.Hash
			if i == 0 {
				entries = append(entries, repository.TreeEntry{ObjectType: repository.Blob, Hash: emptyBlob, Name: "create-clock-1"})
			} else {
				parents = append(parents, head)
			}
			tree, err := repo.StoreTree(ctx, entries)
			require.NoError(t, err)
			commit, err := repo.StoreCommit(ctx, tree, parents...)
			require.NoError(t, err)
			require.NoError(t, repo.UpdateRef(ctx, Namespace, boardId.String(), head, commit))
			head = commit

			for _, want := range vectors.Snapshots {
				if want.AfterPacks != i+1 {
					continue
				}
				b, err := Read(ctx, repo, boardId)
				require.NoError(t, err)
				snap := b.Compile()
				require.Equal(t, want.Title, snap.Title)
				require.Equal(t, want.Description, snap.Description)
				require.Equal(t, want.Columns, columnsOf(snap))
				require.Equal(t, want.Items, itemsOf(snap))
			}
		}
		for _, want := range vectors.Snapshots {
			require.LessOrEqual(t, want.AfterPacks, len(vectors.Packs))
		}

		b, err := Read(ctx, repo, boardId)
		require.NoError(t, err)
		require.NoError(t, b.Validate())
		require.Equal(t, boardId, b.Id())
		require.Equal(t, vectors.Packs[0].Ids["board"], b.Id())

		// every operation is of a type that the board knows, and is written back as
		// it was read
		ops := b.Operations()
		require.Len(t, ops, len(raws))
		for i, op := range ops {
			_, unknown := op.(*dag.UnknownOperation[*Snapshot])
			require.False(t, unknown, raws[i])
			encoded, err := json.Marshal(op)
			require.NoError(t, err)
			require.Equal(t, raws[i], string(encoded))
			require.Equal(t, entity.DeriveId([]byte(raws[i])), op.Id())
		}

		// the ids that the vectors give are the ones of operations of their pack
		for i, pack := range vectors.Packs {
			for what, id := range pack.Ids {
				at, ok := packOf[id]
				require.True(t, ok, what)
				require.Equal(t, i, at, what)
			}
		}
		for _, derivation := range vectors.IdDerivation {
			require.Equal(t, derivation.ExpectedId, entity.DeriveId([]byte(derivation.Raw)))
			require.Contains(t, raws, derivation.Raw)
		}

		// items with the same position are listed by id
		order := slices.Clone(vectors.Order.Items)
		slices.SortFunc(order, func(a, b placed) int {
			return compareOrder(a.Position, a.Id, b.Position, b.Id)
		})
		var ids []entity.Id
		for _, i := range order {
			ids = append(ids, i.Id)
		}
		require.Equal(t, vectors.Order.Order, ids)
	})
}
