package bug

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entity"
)

func TestCommentId(t *testing.T) {
	bugId := entity.Id("abcdefghijklmnopqrstuvwxyz1234__________")
	opId := entity.Id("ABCDEFGHIJ______________________________")
	expectedId := entity.Id("aAbBcCdefDghijEklmnFopqrGstuvHwxyzI1234J")

	mergedId := DeriveCommentId(bugId, opId)
	require.Equal(t, expectedId, mergedId)

	// full length
	splitBugId, splitCommentId := SplitCommentId(mergedId.String())
	require.Equal(t, string(bugId[:30]), splitBugId)
	require.Equal(t, string(opId[:10]), splitCommentId)

	splitBugId, splitCommentId = SplitCommentId(string(expectedId[:6]))
	require.Equal(t, string(bugId[:3]), splitBugId)
	require.Equal(t, string(opId[:3]), splitCommentId)
}
