package util

import (
	"fmt"
	"io"
)

type Hash string

func (h Hash) String() string {
	return string(h)
}

func (h *Hash) UnmarshalGQL(v interface{}) error {
	_, ok := v.(string)
	if !ok {
		return fmt.Errorf("labels must be strings")
	}

	*h = v.(Hash)

	if !h.IsValid() {
		return fmt.Errorf("invalid hash")
	}

	return nil
}

func (h Hash) MarshalGQL(w io.Writer) {
	w.Write([]byte(`"` + h.String() + `"`))
}

func (h *Hash) IsValid() bool {
	if len(*h) != 40 {
		return false
	}
	for _, r := range *h {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
