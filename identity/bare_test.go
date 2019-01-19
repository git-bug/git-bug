package identity

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestBare_Id(t *testing.T) {
	i := NewBare("name", "email")
	id := i.Id()
	assert.Equal(t, "7b226e616d65223a226e616d65222c22", id)
}
