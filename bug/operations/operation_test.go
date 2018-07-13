package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"testing"
)

// Different type with the same fields
type CreateOperation2 struct {
	Title   string
	Message string
}

func (op CreateOperation2) OpType() bug.OperationType {
	return bug.UNKNOW
}

func (op CreateOperation2) Apply(snapshot bug.Snapshot) bug.Snapshot {
	// no-op
	return snapshot
}

func TestOperationsEquality(t *testing.T) {
	var rene = bug.Person{
		Name:  "René Descartes",
		Email: "rene@descartes.fr",
	}

	var A bug.Operation = NewCreateOp(rene, "title", "message")
	var B bug.Operation = NewCreateOp(rene, "title", "message")
	var C bug.Operation = NewCreateOp(rene, "title", "different message")

	if A != B {
		t.Fatal("Equal value ops should be tested equals")
	}

	if A == C {
		t.Fatal("Different value ops should be tested different")
	}

	D := CreateOperation2{Title: "title", Message: "message"}

	if A == D {
		t.Fatal("Operations equality should handle the type")
	}

	var isaac = bug.Person{
		Name:  "Isaac Newton",
		Email: "isaac@newton.uk",
	}

	var E bug.Operation = NewCreateOp(isaac, "title", "message")

	if A == E {
		t.Fatal("Operation equality should handle the author")
	}
}
