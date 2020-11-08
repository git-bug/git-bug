package entity

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// sha-256
const idLength = 64
const humanIdLength = 7

const UnsetId = Id("unset")

// Id is an identifier for an entity or part of an entity
type Id string

// String return the identifier as a string
func (i Id) String() string {
	return string(i)
}

// Human return the identifier, shortened for human consumption
func (i Id) Human() string {
	format := fmt.Sprintf("%%.%ds", humanIdLength)
	return fmt.Sprintf(format, i)
}

func (i Id) HasPrefix(prefix string) bool {
	return strings.HasPrefix(string(i), prefix)
}

// UnmarshalGQL implement the Unmarshaler interface for gqlgen
func (i *Id) UnmarshalGQL(v interface{}) error {
	_, ok := v.(string)
	if !ok {
		return fmt.Errorf("IDs must be strings")
	}

	*i = v.(Id)

	if err := i.Validate(); err != nil {
		return errors.Wrap(err, "invalid ID")
	}

	return nil
}

// MarshalGQL implement the Marshaler interface for gqlgen
func (i Id) MarshalGQL(w io.Writer) {
	_, _ = w.Write([]byte(`"` + i.String() + `"`))
}

// IsValid tell if the Id is valid
func (i Id) Validate() error {
	if len(i) != idLength {
		return fmt.Errorf("invalid length")
	}
	for _, r := range i {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return fmt.Errorf("invalid character")
		}
	}
	return nil
}

/*
 * Sorting
 */

type Alphabetical []Id

func (a Alphabetical) Len() int {
	return len(a)
}

func (a Alphabetical) Less(i, j int) bool {
	return a[i] < a[j]
}

func (a Alphabetical) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
