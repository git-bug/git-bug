package bug

import (
	"encoding/json"
	"testing"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/go-test/deep"
)

func TestOperationPackSerialize(t *testing.T) {
	opp := &OperationPack{}

	opp.Append(createOp)
	opp.Append(setTitleOp)
	opp.Append(addCommentOp)
	opp.Append(setStatusOp)
	opp.Append(labelChangeOp)

	opMeta := NewCreateOp(rene, unix, "title", "message", nil)
	opMeta.SetMetadata("key", "value")
	opp.Append(opMeta)

	if len(opMeta.Metadata) != 1 {
		t.Fatal()
	}

	opFile := NewCreateOp(rene, unix, "title", "message", []git.Hash{
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

	var opp2 *OperationPack
	err = json.Unmarshal(data, &opp2)
	if err != nil {
		t.Fatal(err)
	}

	deep.CompareUnexportedFields = false
	if diff := deep.Equal(opp, opp2); diff != nil {
		t.Fatal(diff)
	}
}
