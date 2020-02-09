package identity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionSerialize(t *testing.T) {
	before := &Version{
		name:      "name",
		email:     "email",
		avatarURL: "avatarUrl",
		keys: []*Key{
			{
				Fingerprint: "fingerprint1",
				PubKey:      "pubkey1",
			},
			{
				Fingerprint: "fingerprint2",
				PubKey:      "pubkey2",
			},
		},
		nonce: makeNonce(20),
		metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		time: 3,
	}

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after Version
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	assert.Equal(t, before, &after)
}
