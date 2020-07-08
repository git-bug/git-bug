package repository

import (
	"fmt"
	"strings"
)

type TreeEntry struct {
	ObjectType ObjectType
	Hash       Hash
	Name       string
}

type ObjectType int

const (
	Unknown ObjectType = iota
	Blob
	Tree
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
	default:
		panic("Unknown git object type")
	}
}

func ParseObjectType(mode, objType string) (ObjectType, error) {
	switch {
	case mode == "100644" && objType == "blob":
		return Blob, nil
	case mode == "040000" && objType == "tree":
		return Tree, nil
	default:
		return Unknown, fmt.Errorf("Unknown git object type %s %s", mode, objType)
	}
}
