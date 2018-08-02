package bug

import (
	"fmt"
	"io"
)

type Label string

func (l Label) String() string {
	return string(l)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (l *Label) UnmarshalGQL(v interface{}) error {
	_, ok := v.(string)
	if !ok {
		return fmt.Errorf("labels must be strings")
	}

	*l = v.(Label)

	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (l Label) MarshalGQL(w io.Writer) {
	w.Write([]byte(`"` + l.String() + `"`))
}
