package todosrht

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidParams(t *testing.T) {
	g := &TodoSourceHut{}
	params := g.ValidParams()

	expected := map[string]interface{}{
		"URL":        nil,
		"BaseURL":    nil,
		"Login":      nil,
		"CredPrefix": nil,
		"Tracker":    nil,
		"TokenRaw":   nil,
	}

	assert.Equal(t, expected, params)
}
