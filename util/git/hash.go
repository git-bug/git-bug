package git

import (
	"fmt"
	"io"
)

const idLengthSHA1 = 40
const idLengthSHA256 = 64

// Hash is a git hash
type Hash string

func (h Hash) String() string {
	return string(h)
}

// UnmarshalGQL implement the Unmarshaler interface for gqlgen
func (h *Hash) UnmarshalGQL(v interface{}) error {
	_, ok := v.(string)
	if !ok {
		return fmt.Errorf("hashes must be strings")
	}

	*h = v.(Hash)

	if !h.IsValid() {
		return fmt.Errorf("invalid hash")
	}

	return nil
}

// MarshalGQL implement the Marshaler interface for gqlgen
func (h Hash) MarshalGQL(w io.Writer) {
	_, _ = w.Write([]byte(`"` + h.String() + `"`))
}

// IsValid tell if the hash is valid
func (h *Hash) IsValid() bool {
	// Support for both sha1 and sha256 git hashes
	if len(*h) != idLengthSHA1 && len(*h) != idLengthSHA256 {
		return false
	}
	for _, r := range *h {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
