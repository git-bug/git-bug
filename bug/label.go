package bug

import (
	"fmt"
	"io"
	"strings"

	"github.com/MichaelMure/git-bug/util/text"
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

func (l Label) Validate() error {
	str := string(l)

	if text.Empty(str) {
		return fmt.Errorf("empty")
	}

	if strings.Contains(str, "\n") {
		return fmt.Errorf("should be a single line")
	}

	if !text.Safe(str) {
		return fmt.Errorf("not fully printable")
	}

	return nil
}
