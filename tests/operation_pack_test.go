package tests

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/MichaelMure/git-bug/bug"
)

func TestOperationPackSerialize(t *testing.T) {
	opp := &bug.OperationPack{}

	opp.Append(createOp)
	opp.Append(setTitleOp)
	opp.Append(addCommentOp)
	opp.Append(setStatusOp)
	opp.Append(labelChangeOp)

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
