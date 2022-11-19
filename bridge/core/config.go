package core

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
)

func FinishConfig(repo *cache.RepoCache, metaKey string, login string) error {
	// if no user exist with the given login metadata
	_, err := repo.ResolveIdentityImmutableMetadata(metaKey, login)
	if err != nil && !entity.IsErrNotFound(err) {
		// real error
		return err
	}
	if err == nil {
		// found an already valid user, all good
		return nil
	}

	// if a default user exist, tag it with the login
	user, err := repo.GetUserIdentity()
	if err != nil && err != identity.ErrNoIdentitySet {
		// real error
		return err
	}
	if err == nil {
		fmt.Printf("Current identity %v tagged with login %v\n", user.Id().Human(), login)
		// found one
		user.SetMetadata(metaKey, login)
		return user.CommitAsNeeded()
	}

	// otherwise create a user with that metadata
	i, err := repo.NewIdentityFromGitUserRaw(map[string]string{
		metaKey: login,
	})
	if err != nil {
		return err
	}

	err = repo.SetUserIdentity(i)
	if err != nil {
		return err
	}

	fmt.Printf("Identity %v created, set as current\n", i.Id().Human())

	return nil
}
