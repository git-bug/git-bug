package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const treeEntryTestHash = Hash("a85730cf5287d40a1e32d3a671ba2296c73387cb")

func TestTreeEntryFormat(t *testing.T) {
	tests := []struct {
		entry    TreeEntry
		expected string
	}{
		{TreeEntry{Blob, treeEntryTestHash, "name"}, "100644 blob a85730cf5287d40a1e32d3a671ba2296c73387cb\tname\n"},
		{TreeEntry{Tree, treeEntryTestHash, "name"}, "040000 tree a85730cf5287d40a1e32d3a671ba2296c73387cb\tname\n"},
		{TreeEntry{Executable, treeEntryTestHash, "name"}, "100755 blob a85730cf5287d40a1e32d3a671ba2296c73387cb\tname\n"},
		{TreeEntry{Symlink, treeEntryTestHash, "name"}, "120000 blob a85730cf5287d40a1e32d3a671ba2296c73387cb\tname\n"},
		{TreeEntry{Submodule, treeEntryTestHash, "name"}, "160000 commit a85730cf5287d40a1e32d3a671ba2296c73387cb\tname\n"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.entry.Format())
		})
	}
}

func TestTreeEntryParse(t *testing.T) {
	tests := []struct {
		line     string
		expected TreeEntry
		err      bool
	}{
		{line: "100644 blob 1e5ffaffc67049635ba7b01f77143313503f1ca1\t.gitignore",
			expected: TreeEntry{Blob, "1e5ffaffc67049635ba7b01f77143313503f1ca1", ".gitignore"}},
		{line: "040000 tree 728421fea4168b874bc1a8aa409d6723ef445a4e\tbug",
			expected: TreeEntry{Tree, "728421fea4168b874bc1a8aa409d6723ef445a4e", "bug"}},
		{line: "100644 blob 1e5ffaffc67049635ba7b01f77143313503f1ca1\tname with  spaces",
			expected: TreeEntry{Blob, "1e5ffaffc67049635ba7b01f77143313503f1ca1", "name with  spaces"}},

		{line: "", err: true},
		{line: "100644 blob 1e5ffaffc67049635ba7b01f77143313503f1ca1", err: true},
		{line: "100644 blob 1e5ffaffc67049635ba7b01f77143313503f1ca1\t", err: true},
		{line: "100644 tree 1e5ffaffc67049635ba7b01f77143313503f1ca1\tname", err: true},
	}

	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			entry, err := ParseTreeEntry(tc.line)
			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, entry)
		})
	}
}

// What the mock repository stores reads back the same.
func TestTreeEntriesRoundTrip(t *testing.T) {
	entries := []TreeEntry{
		{Blob, treeEntryTestHash, "file"},
		{Tree, treeEntryTestHash, "dir"},
		{Executable, treeEntryTestHash, "script.sh"},
		{Symlink, treeEntryTestHash, "link"},
		{Submodule, treeEntryTestHash, "module"},
		{Blob, treeEntryTestHash, "a name with spaces"},
	}

	buffer := prepareTreeEntries(entries)
	read, err := readTreeEntries(buffer.String())
	require.NoError(t, err)
	require.Equal(t, entries, read)
}
