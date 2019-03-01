package identity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdentityStubSerialize(t *testing.T) {
	before := &IdentityStub{
		id: "id1234",
	}

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after IdentityStub
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	assert.Equal(t, before, &after)
}
