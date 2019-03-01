package repository

import (
	"testing"

	"github.com/MichaelMure/git-bug/util/git"
)

func TestTreeEntryFormat(t *testing.T) {

	entries := []TreeEntry{
		{Blob, git.Hash("a85730cf5287d40a1e32d3a671ba2296c73387cb"), "name"},
		{Tree, git.Hash("a85730cf5287d40a1e32d3a671ba2296c73387cb"), "name"},
	}

	for _, entry := range entries {
		_ = entry.Format()
	}
}

func TestTreeEntryParse(t *testing.T) {
	lines := []string{
		"100644 blob 1e5ffaffc67049635ba7b01f77143313503f1ca1	.gitignore",
		"040000 tree 728421fea4168b874bc1a8aa409d6723ef445a4e	bug",
	}

	for _, line := range lines {
		_, err := ParseTreeEntry(line)

		if err != nil {
			t.Fatal(err)
		}
	}

}
