package graphql2

import (
	"encoding/base64"
	"strings"
)


type ResolvedGlobalID struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Takes a type name and an ID specific to that type name, and returns a
// "global ID" that is unique among all types.
func ToGlobalID(ttype string, id string) string {
	str := ttype + ":" + id
	encStr := base64.StdEncoding.EncodeToString([]byte(str))
	return encStr
}

// Takes the "global ID" created by toGlobalID, and returns the type name and ID
// used to create it.
func FromGlobalID(globalID string) *ResolvedGlobalID {
	strID := ""
	b, err := base64.StdEncoding.DecodeString(globalID)
	if err == nil {
		strID = string(b)
	}
	tokens := strings.Split(strID, ":")
	if len(tokens) < 2 {
		return nil
	}
	return &ResolvedGlobalID{
		Type: tokens[0],
		ID:   tokens[1],
	}
}

