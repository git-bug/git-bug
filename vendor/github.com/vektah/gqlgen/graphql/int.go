package graphql

import (
	"fmt"
	"io"
	"strconv"
)

func MarshalInt(i int) Marshaler {
	return WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Itoa(i))
	})
}

func UnmarshalInt(v interface{}) (int, error) {
	switch v := v.(type) {
	case string:
		return strconv.Atoi(v)
	case int:
		return v, nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("%T is not an int", v)
	}
}
