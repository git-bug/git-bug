package identity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MichaelMure/git-bug/entity"
)

func TestBare_Id(t *testing.T) {
	i := NewBare("name", "email")
	id := i.Id()
	expected := entity.Id("e18b853fbd89d5d40ca24811539c9a800c705abd9232f396954e8ca8bb63fa8a")
	assert.Equal(t, expected, id)
}

func TestBareSerialize(t *testing.T) {
	before := &Bare{
		email:     "email",
		name:      "name",
		avatarUrl: "avatar",
	}

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after Bare
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	before.id = after.id

	assert.Equal(t, before, &after)
}
