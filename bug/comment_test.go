package bug

import "testing"

func TestCommentEquality(t *testing.T) {
	c1 := Comment{}
	c2 := Comment{}

	if c1 != c2 {
		t.Fatal()
	}
}
