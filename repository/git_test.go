// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	repo := CreateTestRepo(false)
	defer CleanupTestRepos(t, repo)

	err := repo.LocalConfig().StoreString("section.key", "value")
	assert.NoError(t, err)

	val, err := repo.LocalConfig().ReadString("section.key")
	assert.Equal(t, "value", val)

	err = repo.LocalConfig().StoreString("section.true", "true")
	assert.NoError(t, err)

	val2, err := repo.LocalConfig().ReadBool("section.true")
	assert.Equal(t, true, val2)

	configs, err := repo.LocalConfig().ReadAll("section")
	assert.NoError(t, err)
	assert.Equal(t, configs, map[string]string{
		"section.key":  "value",
		"section.true": "true",
	})

	err = repo.LocalConfig().RemoveAll("section.true")
	assert.NoError(t, err)

	configs, err = repo.LocalConfig().ReadAll("section")
	assert.NoError(t, err)
	assert.Equal(t, configs, map[string]string{
		"section.key": "value",
	})

	_, err = repo.LocalConfig().ReadBool("section.true")
	assert.Equal(t, ErrNoConfigEntry, err)

	err = repo.LocalConfig().RemoveAll("section.nonexistingkey")
	assert.Error(t, err)

	err = repo.LocalConfig().RemoveAll("section.key")
	assert.NoError(t, err)

	_, err = repo.LocalConfig().ReadString("section.key")
	assert.Equal(t, ErrNoConfigEntry, err)

	err = repo.LocalConfig().RemoveAll("nonexistingsection")
	assert.Error(t, err)

	err = repo.LocalConfig().RemoveAll("section")
	assert.Error(t, err)

	_, err = repo.LocalConfig().ReadString("section.key")
	assert.Error(t, err)

	err = repo.LocalConfig().RemoveAll("section.key")
	assert.Error(t, err)
}
