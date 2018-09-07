package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/misc/random_bugs"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
	"github.com/dustin/go-humanize"
	"github.com/ugorji/go/codec"
)

type writer func(opp *bug.OperationPack, repo repository.Repo) (int, util.Hash, error)

type testCase struct {
	name   string
	writer writer
}

func main() {
	packs := random_bugs.GenerateRandomOperationPacks(1000, 5)

	testCases := []testCase{
		{
			name:   "GOB",
			writer: writeGOB,
		},
		{
			name:   "JSON",
			writer: writeJSON,
		},
		{
			name:   "CBOR",
			writer: writeCBOR,
		},
		{
			name:   "MsgPack",
			writer: writeMsgPack,
		},
	}

	for _, testcase := range testCases {
		fmt.Println()
		fmt.Println(testcase.name)

		repo := createRepo(false)

		sizeEmpty, err := dirSize(repo.GetPath())
		if err != nil {
			panic(err)
		}

		// total := int64(0)
		for i, opp := range packs {
			rawSize, hash, err := testcase.writer(opp, repo)
			if err != nil {
				panic(err)
			}

			size := blobSize(hash, repo)

			// total += size

			if i < 10 {
				ratio := float32(size) / float32(rawSize) * 100.0
				fmt.Printf("raw: %v, git: %v, ratio: %v%%\n", rawSize, size, ratio)
			}
		}

		fmt.Println("...")

		sizeFilled, err := dirSize(repo.GetPath())
		if err != nil {
			panic(err)
		}

		err = repo.GC()
		if err != nil {
			panic(err)
		}

		sizePacked, err := dirSize(repo.GetPath())
		if err != nil {
			panic(err)
		}

		err = repo.GCAggressive()
		if err != nil {
			panic(err)
		}

		sizeAggressive, err := dirSize(repo.GetPath())
		if err != nil {
			panic(err)
		}

		fmt.Printf("Unpacked: %v\n", humanize.Bytes(uint64(sizeFilled-sizeEmpty)))
		fmt.Printf("GC packed: %v\n", humanize.Bytes(uint64(sizePacked-sizeEmpty)))
		fmt.Printf("Packing diff: %v\n", sizePacked-sizeFilled)
		fmt.Printf("GC packed aggressive: %v\n", humanize.Bytes(uint64(sizeAggressive-sizeEmpty)))
		fmt.Printf("Packing diff: %v\n", sizeAggressive-sizePacked)
	}
}

func createRepo(bare bool) *repository.GitRepo {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Creating repo:", dir)

	var creator func(string) (*repository.GitRepo, error)

	if bare {
		creator = repository.InitBareGitRepo
	} else {
		creator = repository.InitGitRepo
	}

	repo, err := creator(dir)
	if err != nil {
		log.Fatal(err)
	}

	return repo
}

func writeData(data []byte, repo repository.Repo) (int, util.Hash, error) {
	hash, err := repo.StoreData(data)

	if err != nil {
		return -1, "", err
	}

	return len(data), hash, nil
}

func blobSize(hash util.Hash, repo *repository.GitRepo) int64 {
	rootPath := path.Join(repo.GetPath(), ".git", "objects")

	prefix := hash.String()[:2]
	suffix := hash.String()[2:]

	blobPath := path.Join(rootPath, prefix, suffix)

	fi, err := os.Stat(blobPath)
	if err != nil {
		panic(err)
	}

	return fi.Size()
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func writeGOB(opp *bug.OperationPack, repo repository.Repo) (int, util.Hash, error) {
	data, err := opp.Serialize()
	if err != nil {
		return -1, "", err
	}

	return writeData(data, repo)
}

func writeJSON(opp *bug.OperationPack, repo repository.Repo) (int, util.Hash, error) {
	var data = make([]byte, 0, 64)
	var h codec.Handle = new(codec.JsonHandle)
	var enc = codec.NewEncoderBytes(&data, h)

	err := enc.Encode(opp)
	if err != nil {
		return -1, "", err
	}

	return writeData(data, repo)
}

func writeCBOR(opp *bug.OperationPack, repo repository.Repo) (int, util.Hash, error) {
	var data = make([]byte, 0, 64)
	var h codec.Handle = new(codec.CborHandle)
	var enc = codec.NewEncoderBytes(&data, h)

	err := enc.Encode(opp)
	if err != nil {
		return -1, "", err
	}

	return writeData(data, repo)
}

func writeMsgPack(opp *bug.OperationPack, repo repository.Repo) (int, util.Hash, error) {
	var data = make([]byte, 0, 64)
	var h codec.Handle = new(codec.MsgpackHandle)
	var enc = codec.NewEncoderBytes(&data, h)

	err := enc.Encode(opp)
	if err != nil {
		return -1, "", err
	}

	return writeData(data, repo)
}
