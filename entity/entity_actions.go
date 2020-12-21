package entity

import (
	"fmt"

	"github.com/MichaelMure/git-bug/repository"
)

func ListLocalIds(typename string, repo repository.RepoData) ([]Id, error) {
	refs, err := repo.ListRefs(fmt.Sprintf("refs/%s/", typename))
	if err != nil {
		return nil, err
	}
	return RefsToIds(refs), nil
}

func Fetch() {

}

func Pull() {

}

func Push() {

}

func Remove() error {
	panic("")
}
