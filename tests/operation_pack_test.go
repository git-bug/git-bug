package tests

import (
	"github.com/MichaelMure/git-bug/bug"
	"testing"
)

func TestOperationPackSerialize(t *testing.T) {
	opp := bug.OperationPack{}

	opp.Append(createOp)
	opp.Append(setTitleOp)
	opp.Append(addCommentOp)

	data, err := opp.Serialize()

	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Fatal("empty serialized data")
	}

	_, err = bug.ParseOperationPack(data)

	if err != nil {
		t.Fatal(err)
	}
}
