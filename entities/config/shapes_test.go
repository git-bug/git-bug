package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/util/fracindex"
)

// listItem is the fields of the entries of an ordered list, for the tests: no
// setting is an ordered list yet.
type listItem struct {
	Position Field[string] `json:"position,omitzero"`
	Title    Field[string] `json:"title,omitzero"`
}

func (i listItem) patch(change listItem) listItem {
	i.Position.patch(change.Position)
	i.Title.patch(change.Title)
	return i
}

func (i listItem) Validate() error {
	return i.Position.validate(fracindex.Validate)
}

func (i listItem) position() (string, bool) {
	return i.Position.Get()
}

func TestShapes(t *testing.T) {
	t.Run("absent field", func(t *testing.T) {
		// an absent field is left out of the JSON, as written it would read as cleared
		_, err := json.Marshal(Field[string]{})
		require.Error(t, err)
	})

	t.Run("integer range", func(t *testing.T) {
		accept := func(int64) error { return nil }
		for _, value := range []int64{0, maxInteger, -maxInteger} {
			require.NoError(t, SetField(value).validate(accept))
		}
		for _, value := range []int64{maxInteger + 1, -maxInteger - 1} {
			require.Error(t, SetField(value).validate(accept))
		}
	})

	t.Run("ordered list", func(t *testing.T) {
		var l list[string, listItem]
		set := func(key string, change listItem) {
			(&setEntry[string, listItem]{Key: key, Fields: change}).apply(&l)
		}
		remove := func(key string) {
			(&removeEntry[string, listItem]{Key: key}).apply(&l)
		}

		set("first", listItem{Position: SetField("V"), Title: SetField("first")})
		set("second", listItem{Position: SetField("k")})
		set("third", listItem{Position: SetField("t")})
		set("unplaced b", listItem{})
		set("unplaced a", listItem{})
		// the same position as second, as two concurrent inserts at the same spot
		set("also second", listItem{Position: SetField("k")})
		require.Equal(t, []string{"first", "also second", "second", "third", "unplaced a", "unplaced b"}, ordered(l))

		// moving an entry changes only its position
		set("first", listItem{Position: SetField("x")})
		require.Equal(t, []string{"also second", "second", "third", "first", "unplaced a", "unplaced b"}, ordered(l))
		require.Equal(t, listItem{Position: SetField("x"), Title: SetField("first")}, l["first"])

		// an entry without position anymore comes last
		set("third", listItem{Position: ClearField[string]()})
		remove("also second")
		require.Equal(t, []string{"second", "first", "third", "unplaced a", "unplaced b"}, ordered(l))
	})
}
