package entity

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const IdLengthSHA1 = 40
const IdLengthSHA256 = 64
const humanIdLength = 7

const UnsetId = Id("unset")

// Id is an identifier for an entity or part of an entity
type Id string

func (i Id) String() string {
	return string(i)
}

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
	if len(i) != IdLengthSHA1 && len(i) != IdLengthSHA256 {
		return fmt.Errorf("invalid length")
	}
	for _, r := range i {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return fmt.Errorf("invalid character")
		}
	}
	return nil
}
