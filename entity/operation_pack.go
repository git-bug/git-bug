package entity

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

const opsEntryName = "ops"
const versionEntryPrefix = "version-"
const createClockEntryPrefix = "create-clock-"
const editClockEntryPrefix = "edit-clock-"

type operationPack struct {
	Operations []Operation
	CreateTime lamport.Time
	EditTime   lamport.Time
}

// func (opp *operationPack) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(struct {
// 		Operations []Operation `json:"ops"`
// 	}{
// 		Operations: opp.Operations,
// 	})
// }

func readOperationPack(def Definition, repo repository.RepoData, treeHash repository.Hash) (*operationPack, error) {
	entries, err := repo.ReadTree(treeHash)
	if err != nil {
		return nil, err
	}

	// check the format version first, fail early instead of trying to read something
	// var version uint
	// for _, entry := range entries {
	// 	if strings.HasPrefix(entry.Name, versionEntryPrefix) {
	// 		v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, versionEntryPrefix), 10, 64)
	// 		if err != nil {
	// 			return nil, errors.Wrap(err, "can't read format version")
	// 		}
	// 		if v > 1<<12 {
	// 			return nil, fmt.Errorf("format version too big")
	// 		}
	// 		version = uint(v)
	// 		break
	// 	}
	// }
	// if version == 0 {
	// 	return nil, NewErrUnknowFormat(def.formatVersion)
	// }
	// if version != def.formatVersion {
	// 	return nil, NewErrInvalidFormat(version, def.formatVersion)
	// }

	var ops []Operation
	var createTime lamport.Time
	var editTime lamport.Time

	for _, entry := range entries {
		if entry.Name == opsEntryName {
			data, err := repo.ReadData(entry.Hash)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read git blob data")
			}

			ops, err = unmarshallOperations(def, data)
			if err != nil {
				return nil, err
			}
			break
		}

		if strings.HasPrefix(entry.Name, createClockEntryPrefix) {
			v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, createClockEntryPrefix), 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "can't read creation lamport time")
			}
			createTime = lamport.Time(v)
		}

		if strings.HasPrefix(entry.Name, editClockEntryPrefix) {
			v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, editClockEntryPrefix), 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "can't read edit lamport time")
			}
			editTime = lamport.Time(v)
		}
	}

	return &operationPack{
		Operations: ops,
		CreateTime: createTime,
		EditTime:   editTime,
	}, nil
}

func unmarshallOperations(def Definition, data []byte) ([]Operation, error) {
	aux := struct {
		Operations []json.RawMessage `json:"ops"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return nil, err
	}

	ops := make([]Operation, 0, len(aux.Operations))

	for _, raw := range aux.Operations {
		// delegate to specialized unmarshal function
		op, err := def.operationUnmarshaler(raw)
		if err != nil {
			return nil, err
		}

		ops = append(ops, op)
	}

	return ops, nil
}
