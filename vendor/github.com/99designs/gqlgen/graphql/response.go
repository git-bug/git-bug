package graphql

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vektah/gqlparser/gqlerror"
)

type Response struct {
	Data   json.RawMessage `json:"data"`
	Errors gqlerror.List   `json:"errors,omitempty"`
}

func ErrorResponse(ctx context.Context, messagef string, args ...interface{}) *Response {
	return &Response{
		Errors: gqlerror.List{{Message: fmt.Sprintf(messagef, args...)}},
	}
}
