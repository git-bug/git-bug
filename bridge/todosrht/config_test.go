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

func TestValidateBaseURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"https://todo.sr.ht", "https://todo.sr.ht", true},
		{"https://todo.sr.ht/", "https://todo.sr.ht", true},
		{"http://localhost:5003", "http://localhost:5003", true},
		{"http://127.0.0.1:5003", "http://127.0.0.1:5003", true},
		{"http://[::1]:5003", "http://[::1]:5003", true},
		{"http://todo.sr.ht", "", false},
		{"ftp://todo.sr.ht", "", false},
		{"todo.sr.ht", "", false},
		{"https://", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := validateBaseURL(tt.in)
			if !tt.ok {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseTodoURLRejectsPlainHTTP(t *testing.T) {
	_, _, err := parseTodoURL("http://todo.sr.ht/~owner/tracker")
	assert.Error(t, err)

	base, tracker, err := parseTodoURL("https://todo.sr.ht/~owner/tracker")
	assert.NoError(t, err)
	assert.Equal(t, "https://todo.sr.ht", base)
	assert.Equal(t, "tracker", tracker)
}
