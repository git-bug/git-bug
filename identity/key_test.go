package identity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeKeyFingerprint(t *testing.T) {
	checkEncodeDecodeKeyFingerprint(t, strings.Repeat("0", 40))
	checkEncodeDecodeKeyFingerprint(t, strings.Repeat("E", 40))
	checkEncodeDecodeKeyFingerprint(t, "C77E1D7542889EC0E45BA88899DA3BE167DA2410")
}

func checkEncodeDecodeKeyFingerprint(t *testing.T, fingerprint string) {
	decoded, err := DecodeKeyFingerprint(fingerprint)
	require.NoError(t, err)
	require.Equal(t, fingerprint, EncodeKeyFingerprint(decoded))
}

