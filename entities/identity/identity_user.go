package identity

import (
	"fmt"
	"os"

	"github.com/git-bug/gitconfig"
	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// SetUserIdentity store the user identity's id in the git config
func SetUserIdentity(repo repository.RepoConfig, identity *Identity) error {
	return repo.Config().Update(func(f *gitconfig.File) error {
		return f.ReplaceAll(identityConfigKey, identity.Id().String())
	})
}

func ClearUserIdentity(repo repository.RepoConfig) error {
	return repo.Config().Update(func(f *gitconfig.File) error {
		_, err := f.UnsetAll(identityConfigKey)
		return err
	})
}

// GetUserIdentity read the current user identity, set with a git config entry
func GetUserIdentity(repo repository.Repo) (*Identity, error) {
	id, err := GetUserIdentityId(repo)
	if err != nil {
		return nil, err
	}

	i, err := Read(repo, id)
	if entity.IsErrNotFound(err) {
		innerErr := ClearUserIdentity(repo)
		if innerErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, errors.Wrap(innerErr, "can't clear user identity").Error())
		}
		return nil, err
	}

	return i, nil
}

func GetUserIdentityId(repo repository.Repo) (entity.Id, error) {
	cfg, err := repo.Config().Read()
	if err != nil {
		return entity.UnsetId, err
	}
	entries := cfg.GetAll(identityConfigKey)
	switch {
	case len(entries) == 0:
		return entity.UnsetId, ErrNoIdentitySet
	case len(entries) > 1:
		return entity.UnsetId, ErrMultipleIdentitiesSet
	}

	var id = entity.Id(entries[0].Value)

	if err := id.Validate(); err != nil {
		return entity.UnsetId, err
	}

	return id, nil
}

// IsUserIdentitySet say if the user has set his identity
func IsUserIdentitySet(repo repository.Repo) (bool, error) {
	cfg, err := repo.Config().Read()
	if err != nil {
		return false, err
	}
	switch len(cfg.GetAll(identityConfigKey)) {
	case 0:
		return false, nil
	case 1:
		return true, nil
	}
	return false, ErrMultipleIdentitiesSet
}
