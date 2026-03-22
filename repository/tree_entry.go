package repository

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type TreeEntry struct {
	ObjectType ObjectType
	Hash       Hash
	Name       string
}

type ObjectType int

const (
	Unknown    ObjectType = iota
	Blob                  // regular file      (100644)
	Tree                  // directory         (040000)
	Executable            // executable file   (100755)
	Symlink               // symbolic link     (120000)
	Submodule             // git submodule     (160000)
)

func ParseTreeEntry(line string) (TreeEntry, error) {
	fields := strings.Fields(line)

	if len(fields) < 4 {
		return TreeEntry{}, fmt.Errorf("Invalid input to parse as a TreeEntry")
	}

	objType, err := ParseObjectType(fields[0], fields[1])

	if err != nil {
		return TreeEntry{}, err
	}

	hash := Hash(fields[2])
	name := strings.Join(fields[3:], "")

	return TreeEntry{
		ObjectType: objType,
		Hash:       hash,
		Name:       name,
	}, nil
}

// Format the entry as a git ls-tree compatible line
func (entry TreeEntry) Format() string {
	return fmt.Sprintf("%s %s\t%s\n", entry.ObjectType.Format(), entry.Hash, entry.Name)
}

func (ot ObjectType) Format() string {
	switch ot {
	case Blob:
		return "100644 blob"
	case Tree:
		return "040000 tree"
	case Executable:
		return "100755 blob"
	case Symlink:
		return "120000 blob"
	case Submodule:
		return "160000 commit"
	default:
		panic("Unknown git object type")
	}
}

func (ot ObjectType) MarshalGQL(w io.Writer) {
	switch ot {
	case Tree:
		fmt.Fprint(w, strconv.Quote("TREE"))
	case Blob, Executable:
		fmt.Fprint(w, strconv.Quote("BLOB"))
	case Symlink:
		fmt.Fprint(w, strconv.Quote("SYMLINK"))
	case Submodule:
		fmt.Fprint(w, strconv.Quote("SUBMODULE"))
	default:
		panic(fmt.Sprintf("unknown ObjectType value %d", int(ot)))
	}
}

func (ot *ObjectType) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	switch str {
	case "TREE":
		*ot = Tree
	case "BLOB":
		*ot = Blob
	case "SYMLINK":
		*ot = Symlink
	case "SUBMODULE":
		*ot = Submodule
	default:
		return fmt.Errorf("%q is not a valid ObjectType", str)
	}
	return nil
}

func ParseObjectType(mode, objType string) (ObjectType, error) {
	switch {
	case mode == "100644" && objType == "blob":
		return Blob, nil
	case mode == "040000" && objType == "tree":
		return Tree, nil
	case mode == "100755" && objType == "blob":
		return Executable, nil
	case mode == "120000" && objType == "blob":
		return Symlink, nil
	case mode == "160000" && objType == "commit":
		return Submodule, nil
	default:
		return Unknown, fmt.Errorf("Unknown git object type %s %s", mode, objType)
	}
}

func prepareTreeEntries(entries []TreeEntry) bytes.Buffer {
	var buffer bytes.Buffer

	for _, entry := range entries {
		buffer.WriteString(entry.Format())
	}

	return buffer
}

func readTreeEntries(s string) ([]TreeEntry, error) {
	split := strings.Split(strings.TrimSpace(s), "\n")

	casted := make([]TreeEntry, len(split))
	for i, line := range split {
		if line == "" {
			continue
		}

		entry, err := ParseTreeEntry(line)

		if err != nil {
			return nil, err
		}

		casted[i] = entry
	}

	return casted, nil
}

// SearchTreeEntry search a TreeEntry by name from an array
func SearchTreeEntry(entries []TreeEntry, name string) (TreeEntry, bool) {
	for _, entry := range entries {
		if entry.Name == name {
			return entry, true
		}
	}
	return TreeEntry{}, false
}
