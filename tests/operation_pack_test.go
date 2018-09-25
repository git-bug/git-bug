package tests

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/operations"
	"github.com/MichaelMure/git-bug/util/git"
)

func TestOperationPackSerialize(t *testing.T) {
	opp := &bug.OperationPack{}

	opp.Append(createOp)
	opp.Append(setTitleOp)
	opp.Append(addCommentOp)
	opp.Append(setStatusOp)
	opp.Append(labelChangeOp)

	opMeta := operations.NewCreateOp(rene, unix, "title", "message", nil)
	opMeta.SetMetadata("key", "value")
	opp.Append(opMeta)

	if len(opMeta.Metadata) != 1 {
		t.Fatal()
	}

	opFile := operations.NewCreateOp(rene, unix, "title", "message", []git.Hash{
		"abcdef",
		"ghijkl",
	})
	opp.Append(opFile)

	if len(opFile.Files) != 2 {
		t.Fatal()
	}

	data, err := json.Marshal(opp)
	if err != nil {
		t.Fatal(err)
	}

	var opp2 *bug.OperationPack
	err = json.Unmarshal(data, &opp2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(opp, opp2) {
		t.Fatalf("%v and %v are different", opp, opp2)
	}
}
