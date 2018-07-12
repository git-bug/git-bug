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

func (op CreateOperation2) OpType() OperationType {
	return UNKNOW
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

	var A Operation = NewCreateOp(rene, "title", "message")
	var B Operation = NewCreateOp(rene, "title", "message")
	var C Operation = NewCreateOp(rene, "title", "different message")

	if A != B {
		t.Fatal("Equal value operations should be tested equals")
	}

	if A == C {
		t.Fatal("Different value operations should be tested different")
	}

	D := CreateOperation2{Title: "title", Message: "message"}

	if A == D {
		t.Fatal("Operations equality should handle the type")
	}

	var isaac = bug.Person{
		Name:  "Isaac Newton",
		Email: "isaac@newton.uk",
	}

	var E Operation = NewCreateOp(isaac, "title", "message")

	if A == E {
		t.Fatal("Operation equality should handle the author")
	}
}
