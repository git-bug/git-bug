package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"testing"
)

func TestOperationPackSerialize(t *testing.T) {
	opp := bug.OperationPack{}

	opp.Append(createOp)
	opp.Append(setTitleOp)
	opp.Append(addCommentOp)

	jsonBytes, err := opp.Serialize()

	if err != nil {
		t.Fatal(err)
	}

	if len(jsonBytes) == 0 {
		t.Fatal("empty json")
	}

	fmt.Println(prettyPrintJSON(jsonBytes))
}

func prettyPrintJSON(jsonBytes []byte) (string, error) {
	var prettyBytes bytes.Buffer
	err := json.Indent(&prettyBytes, jsonBytes, "", "  ")
	if err != nil {
		return "", err
	}
	return prettyBytes.String(), nil
}
