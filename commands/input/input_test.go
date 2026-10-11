package input

import (
	"testing"

	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/repository"
)

func TestLaunchEditorWithTemplateInProgress(t *testing.T) {
	repo := repository.NewMockRepo()

	err := util.WriteFile(repo.LocalStorage(), "TEST_EDITMSG", []byte("other message"), 0644)
	require.NoError(t, err)

	_, err = LaunchEditorWithTemplate(repo, "TEST_EDITMSG", "template")
	require.ErrorContains(t, err, "file exists")

	// the other edit's file is left untouched
	content, err := util.ReadFile(repo.LocalStorage(), "TEST_EDITMSG")
	require.NoError(t, err)
	require.Equal(t, "other message", string(content))
}
