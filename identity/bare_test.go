package identity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBare_Id(t *testing.T) {
	i := NewBare("name", "email")
	id := i.Id()
	assert.Equal(t, "7b226e616d65223a226e616d65222c22", id)
}

func TestBareSerialize(t *testing.T) {
	before := &Bare{
		login:     "login",
		email:     "email",
		name:      "name",
		avatarUrl: "avatar",
	}

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after Bare
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	assert.Equal(t, before, &after)
}
