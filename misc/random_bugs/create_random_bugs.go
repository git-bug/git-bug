package random_bugs

import (
	"math/rand"
	"strings"
	"time"

	"github.com/icrowley/fake"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/repository"
)

type opsGenerator func(bug.ReadWrite, identity.Interface, int64)

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

func FillRepo(repo repository.ClockedRepo, bugNumber int) {
	FillRepoWithSeed(repo, bugNumber, time.Now().UnixNano())
}

func FillRepoWithSeed(repo repository.ClockedRepo, bugNumber int, seed int64) {
	options := DefaultOptions()
	options.BugNumber = bugNumber

	CommitRandomBugsWithSeed(repo, options, seed)
}

func CommitRandomBugs(repo repository.ClockedRepo, opts Options) {
	CommitRandomBugsWithSeed(repo, opts, time.Now().UnixNano())
}

func CommitRandomBugsWithSeed(repo repository.ClockedRepo, opts Options, seed int64) {
	generateRandomPersons(repo, opts.PersonNumber)

	bugs := generateRandomBugsWithSeed(opts, seed)

	for _, b := range bugs {
		err := b.Commit(repo)
		if err != nil {
			panic(err)
		}
	}
}

func generateRandomBugsWithSeed(opts Options, seed int64) []*bug.Bug {
	rand.Seed(seed)
	fake.Seed(seed)

	// At the moment git-bug has a risk of hash collision is simple
	// operation (like open/close) are made with the same timestamp.
	// As a temporary workaround, we use here an strictly increasing
	// timestamp
	timestamp := time.Now().Unix()

	opsGenerators := []opsGenerator{
		comment,
		comment,
		title,
		labels,
		open,
		close,
	}

	result := make([]*bug.Bug, opts.BugNumber)

	for i := 0; i < opts.BugNumber; i++ {
		addedLabels = []string{}

		b, _, err := bug.Create(
			randomPerson(),
			time.Now().Unix(),
			fake.Sentence(),
			paragraphs(),
			nil, nil,
		)

		if err != nil {
			panic(err)
		}

		nOps := opts.MinOp

		if opts.MaxOp > opts.MinOp {
			nOps += rand.Intn(opts.MaxOp - opts.MinOp)
		}

		for j := 0; j < nOps; j++ {
			index := rand.Intn(len(opsGenerators))
			opsGenerators[index](b, randomPerson(), timestamp)
			timestamp++
		}

		result[i] = b
	}

	return result
}

func person(repo repository.RepoClock) (*identity.Identity, error) {
	return identity.NewIdentity(repo, fake.FullName(), fake.EmailAddress())
}

var persons []*identity.Identity

func generateRandomPersons(repo repository.ClockedRepo, n int) {
	persons = make([]*identity.Identity, n)
	for i := range persons {
		p, err := person(repo)
		if err != nil {
			panic(err)
		}
		err = p.Commit(repo)
		if err != nil {
			panic(err)
		}
		persons[i] = p
	}
}

func randomPerson() identity.Interface {
	index := rand.Intn(len(persons))
	return persons[index]
}

func paragraphs() string {
	p := fake.Paragraphs()
	return strings.Replace(p, "\t", "\n\n", -1)
}

func comment(b bug.ReadWrite, p identity.Interface, timestamp int64) {
	_, _, _ = bug.AddComment(b, p, timestamp, paragraphs(), nil, nil)
}

func title(b bug.ReadWrite, p identity.Interface, timestamp int64) {
	_, _ = bug.SetTitle(b, p, timestamp, fake.Sentence(), nil)
}

func open(b bug.ReadWrite, p identity.Interface, timestamp int64) {
	_, _ = bug.Open(b, p, timestamp, nil)
}

func close(b bug.ReadWrite, p identity.Interface, timestamp int64) {
	_, _ = bug.Close(b, p, timestamp, nil)
}

var addedLabels []string

func labels(b bug.ReadWrite, p identity.Interface, timestamp int64) {
	var removed []string
	nbRemoved := rand.Intn(3)
	for nbRemoved > 0 && len(addedLabels) > 0 {
		index := rand.Intn(len(addedLabels))
		removed = append(removed, addedLabels[index])
		addedLabels[index] = addedLabels[len(addedLabels)-1]
		addedLabels = addedLabels[:len(addedLabels)-1]
		nbRemoved--
	}

	var added []string
	nbAdded := rand.Intn(3)
	for i := 0; i < nbAdded; i++ {
		label := fake.Word()
		added = append(added, label)
		addedLabels = append(addedLabels, label)
	}

	// ignore error
	// if the randomisation produce no changes, no op
	// is added to the bug
	_, _, _ = bug.ChangeLabels(b, p, timestamp, added, removed, nil)
}
