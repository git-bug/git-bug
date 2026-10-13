package core

import (
	"fmt"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
)

func FinishConfig(repo *cache.RepoCache, metaKey string, login string) error {
	// Check the configured user identity first so dangling identities are handled.
	user, err := repo.GetUserIdentity()
	danglingUser := false
	switch {
	case err == nil, err == identity.ErrNoIdentitySet:
	case entity.IsErrNotFound(err):
		// The configured user identity points to an identity that doesn't
		// exist (anymore) in this repository. Don't fail the whole bridge
		// configuration for that, just create or adopt a valid identity below,
		// which will replace the dangling one as the current user.
		fmt.Printf("The configured user identity doesn't exist in this repository, ignoring it\n")
		danglingUser = true
	default:
		// real error
		return err
	}

	// if a user exists with the given login metadata
	existing, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKey, login)
	if err != nil && !entity.IsErrNotFound(err) {
		// real error
		return err
	}
	if err == nil {
		// found an already valid user
		if danglingUser {
			if err := repo.SetUserIdentity(existing); err != nil {
				return err
			}
			fmt.Printf("Identity %v set as current\n", existing.Id().Human())
		}
		return nil
	}

	// if a default user exist, tag it with the login
	if user != nil {
		fmt.Printf("Current identity %v tagged with login %v\n", user.Id().Human(), login)
		// found one
		user.SetMetadata(metaKey, login)
		return user.CommitAsNeeded()
	}

	// otherwise create a user with that metadata
	i, err := repo.Identities().NewFromGitUserRaw(map[string]string{
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
