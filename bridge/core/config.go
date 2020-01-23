package core

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
)

func FinishConfig(repo *cache.RepoCache, metaKey string, login string) error {
	// if no user exist with the given login metadata
	_, err := repo.ResolveIdentityImmutableMetadata(metaKey, login)
	if err != nil && err != identity.ErrIdentityNotExist {
		// real error
		return err
	}
	if err == nil {
		// found an already valid user, all good
		return nil
	}

	// if a default user exist, tag it with the login
	user, err := repo.GetUserIdentity()
	if err != nil && err != identity.ErrIdentityNotExist {
		// real error
		return err
	}
	if err == nil {
		// found one
		user.SetMetadata(metaKey, login)
		return user.CommitAsNeeded()
	}

	// otherwise create a user with that metadata
	i, err := repo.NewIdentityFromGitUserRaw(map[string]string{
		metaKey: login,
	})

	err = repo.SetUserIdentity(i)
	if err != nil {
		return err
	}

	return nil
}
