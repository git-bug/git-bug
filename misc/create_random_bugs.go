// +build ignore

package main

import (
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/icrowley/fake"
)

const bugNumber = 40
const personNumber = 5
const minOp = 3
const maxOp = 40

type opsGenerator func(*bug.Bug, bug.Person)

// This program will randomly generate a collection of bugs in the repository
// of the current path
func main() {
	rand.Seed(time.Now().UnixNano())

	opsGenerators := []opsGenerator{
		comment,
		comment,
		title,
		labels,
		operations.Open,
		operations.Close,
	}

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	repo, err := repository.NewGitRepo(dir, func(repo *repository.GitRepo) error {
		return nil
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < bugNumber; i++ {
		addedLabels = []string{}

		b, err := operations.Create(randomPerson(), fake.Sentence(), paragraphs())

		if err != nil {
			panic(err)
		}

		nOps := minOp + rand.Intn(maxOp-minOp)
		for j := 0; j < nOps; j++ {
			index := rand.Intn(len(opsGenerators))
			opsGenerators[index](b, randomPerson())
		}

		err = b.Commit(repo)
		if err != nil {
			panic(err)
		}
	}
}

func person() bug.Person {
	return bug.Person{
		Name:  fake.FullName(),
		Email: fake.EmailAddress(),
	}
}

var persons []bug.Person

func randomPerson() bug.Person {
	if len(persons) == 0 {
		persons = make([]bug.Person, personNumber)
		for i := range persons {
			persons[i] = person()
		}
	}

	index := rand.Intn(personNumber)
	return persons[index]
}

func paragraphs() string {
	p := fake.Paragraphs()
	return strings.Replace(p, "\t", "\n\n", -1)
}

func comment(b *bug.Bug, p bug.Person) {
	operations.Comment(b, p, paragraphs())
}

func title(b *bug.Bug, p bug.Person) {
	operations.SetTitle(b, p, fake.Sentence())
}

var addedLabels []string

func labels(b *bug.Bug, p bug.Person) {
	var removed []string
	nbRemoved := rand.Intn(3)
	for nbRemoved > 0 && len(addedLabels) > 0 {
		index := rand.Intn(len(addedLabels))
		removed = append(removed, addedLabels[index])
		addedLabels[index] = addedLabels[len(addedLabels)-1]
		addedLabels = addedLabels[:len(addedLabels)-1]
	}

	var added []string
	nbAdded := rand.Intn(3)
	for i := 0; i < nbAdded; i++ {
		label := fake.Word()
		added = append(added, label)
		addedLabels = append(addedLabels, label)
	}

	operations.ChangeLabels(nil, b, p, added, removed)
}
