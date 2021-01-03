package identity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyJSON(t *testing.T) {
	k := generatePublicKey()

	data, err := json.Marshal(k)
	require.NoError(t, err)

	var read Key
	err = json.Unmarshal(data, &read)
	require.NoError(t, err)

	require.Equal(t, k, &read)
}
