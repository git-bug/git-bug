package random_bugs

import (
	"math/rand"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/icrowley/fake"
)

type opsGenerator func(bug.Interface, bug.Person)

type Options struct {
	BugNumber    int
	PersonNumber int
	MinOp        int
	MaxOp        int
}

func DefaultOptions() Options {
	return Options{
		BugNumber:    15,
		PersonNumber: 5,
		MinOp:        3,
		MaxOp:        20,
	}
}

func CommitRandomBugs(repo repository.Repo, opts Options) {
	CommitRandomBugsWithSeed(repo, opts, time.Now().UnixNano())
}

func CommitRandomBugsWithSeed(repo repository.Repo, opts Options, seed int64) {
	bugs := GenerateRandomBugsWithSeed(opts, seed)

	for _, b := range bugs {
		err := b.Commit(repo)
		if err != nil {
			panic(err)
		}
	}
}

func GenerateRandomBugs(opts Options) []*bug.Bug {
	return GenerateRandomBugsWithSeed(opts, time.Now().UnixNano())
}

func GenerateRandomBugsWithSeed(opts Options, seed int64) []*bug.Bug {
	rand.Seed(seed)
	fake.Seed(seed)

	opsGenerators := []opsGenerator{
		comment,
		comment,
		title,
		labels,
		operations.Open,
		operations.Close,
	}

	result := make([]*bug.Bug, opts.BugNumber)

	for i := 0; i < opts.BugNumber; i++ {
		addedLabels = []string{}

		b, err := operations.Create(randomPerson(opts.PersonNumber), fake.Sentence(), paragraphs())

		if err != nil {
			panic(err)
		}

		nOps := opts.MinOp + rand.Intn(opts.MaxOp-opts.MinOp)
		for j := 0; j < nOps; j++ {
			index := rand.Intn(len(opsGenerators))
			opsGenerators[index](b, randomPerson(opts.PersonNumber))
		}

		result[i] = b
	}

	return result
}

func GenerateRandomOperationPacks(packNumber int, opNumber int) []*bug.OperationPack {
	return GenerateRandomOperationPacksWithSeed(packNumber, opNumber, time.Now().UnixNano())
}

func GenerateRandomOperationPacksWithSeed(packNumber int, opNumber int, seed int64) []*bug.OperationPack {
	// Note: this is a bit crude, only generate a Create + Comments

	rand.Seed(seed)
	fake.Seed(seed)

	result := make([]*bug.OperationPack, packNumber)

	for i := 0; i < packNumber; i++ {
		opp := &bug.OperationPack{}

		var op bug.Operation

		op = operations.NewCreateOp(randomPerson(5), fake.Sentence(), paragraphs(), nil)

		opp.Append(op)

		for j := 0; j < opNumber-1; j++ {
			op = operations.NewAddCommentOp(randomPerson(5), paragraphs(), nil)
			opp.Append(op)
		}

		result[i] = opp
	}

	return result
}

func person() bug.Person {
	return bug.Person{
		Name:  fake.FullName(),
		Email: fake.EmailAddress(),
	}
}

var persons []bug.Person

func randomPerson(personNumber int) bug.Person {
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

func comment(b bug.Interface, p bug.Person) {
	operations.Comment(b, p, paragraphs())
}

func title(b bug.Interface, p bug.Person) {
	operations.SetTitle(b, p, fake.Sentence())
}

var addedLabels []string

func labels(b bug.Interface, p bug.Person) {
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
