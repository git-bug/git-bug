package identity

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeKeyFingerprint(t *testing.T) {
	checkEncodeDecodeKeyFingerprint(t, strings.Repeat("0", 40))
	checkEncodeDecodeKeyFingerprint(t, strings.Repeat("E", 40))
	checkEncodeDecodeKeyFingerprint(t, "C77E1D7542889EC0E45BA88899DA3BE167DA2410")
}

func checkEncodeDecodeKeyFingerprint(t *testing.T, fingerprint string) {
	decoded, err := decodeKeyFingerprint(fingerprint)
	require.NoError(t, err)
	require.Equal(t, fingerprint, encodeKeyFingerprint(decoded))
}

func TestKeySerialize(t *testing.T) {
	armored := `-----BEGIN PGP PUBLIC KEY BLOCK-----

mI0EXrVFzwEEAL7roW5pBs7PYhnW8XdHuMBUrOqx+TR8JPsTLzlFFKHniJ7Cxm24
rj+nCiVAC3yI5hEWbLYLp6HoSCAEJim+ac+LsoH0Rxz325l+EYz7nAq44rebfNuy
A5LD9/KVzLAu0FO27pgCiH9RpsFVYveHYtR1jDDvag6MLdlTZaQfqCGnABEBAAG0
LFJlbsOpIERlc2NhcnRlcyA8cmVuZS5kZXNjYXJ0ZXNAZXhhbXBsZS5jb20+iM4E
EwEKADgWIQQpwni46BlhwjZt/3bXoSG7jO2rwwUCXrVFzwIbAwULCQgHAgYVCgkI
CwIEFgIDAQIeAQIXgAAKCRDXoSG7jO2rw5LcBACPp+2cwUcYCiHZVUAnAHokY7R0
msjA/YryCy+rcW86TcV7KuyZjg3wCHi7DrDuvYIDXr83W2XaCoJktAW/+aENj8QH
6r7Tini3ENmNT8caqiCJGE0iY0QRXZomxAoZc5kvDq596ifoUA08ILncGla7Uq04
+3Da+JBLWoDQvVP3xbiNBF61Rc8BBADBYKVgB1eHgXOorCeKYLCDSNwklkkdCN5u
WZygmr/EMpT7YGuAvW70WKHcd0zo+MX/3fWvJeTQDVmReNF0zJv0OSjcAsamcOQ9
G9rdL3bKWMGJRtTeXmtZ6BkP4YU227VkFTFXvQzt8WD5CXGQJtEZRXQqHKNjNNIY
JUxF6VfJtQARAQABiLYEGAEKACAWIQQpwni46BlhwjZt/3bXoSG7jO2rwwUCXrVF
zwIbDAAKCRDXoSG7jO2rw7xEA/9TJD/M6vO160zNr7fCB31rqGUvkHWOKaeSHJmG
AvFBrNiBG+nGRjc2XbZqSaykO7ApcmLzgh8zzlB3gxZjorbEGRoEUsYD5pmZhfFl
kZyE/aXEbuTIXXcR9fyuDGvP2eF4RPth8P4ew9ycXl89IUdbapD3JKg/ptkgw8dy
y8TVdw==
=01qL
-----END PGP PUBLIC KEY BLOCK-----`

	before, err := NewKeyFromArmored(armored)
	assert.NoError(t, err)
	assert.NoError(t, before.Validate())

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after Key
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)
	assert.NoError(t, after.Validate())

	assert.NotEmpty(t, after.Fingerprint())
	assert.Equal(t, before.Fingerprint(), after.Fingerprint())
}
