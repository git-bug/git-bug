// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	repo := CreateTestRepo(false)
	defer CleanupTestRepos(t, repo)

	err := repo.StoreConfig("section.key", "value")
	assert.NoError(t, err)

	val, err := repo.ReadConfigString("section.key")
	assert.Equal(t, "value", val)

	err = repo.StoreConfig("section.true", "true")
	assert.NoError(t, err)

	val2, err := repo.ReadConfigBool("section.true")
	assert.Equal(t, true, val2)

	configs, err := repo.ReadConfigs("section")
	assert.NoError(t, err)
	assert.Equal(t, configs, map[string]string{
		"section.key":  "value",
		"section.true": "true",
	})

	err = repo.RmConfigs("section.true")
	assert.NoError(t, err)

	configs, err = repo.ReadConfigs("section")
	assert.NoError(t, err)
	assert.Equal(t, configs, map[string]string{
		"section.key": "value",
	})

	_, err = repo.ReadConfigBool("section.true")
	assert.Equal(t, ErrNoConfigEntry, err)

	err = repo.RmConfigs("section.nonexistingkey")
	assert.Error(t, err)

	err = repo.RmConfigs("section.key")
	assert.NoError(t, err)

	_, err = repo.ReadConfigString("section.key")
	assert.Equal(t, ErrNoConfigEntry, err)

	err = repo.RmConfigs("nonexistingsection")
	assert.Error(t, err)

	err = repo.RmConfigs("section")
	assert.NoError(t, err)

	_, err = repo.ReadConfigString("section.key")
	assert.Error(t, err)

	err = repo.RmConfigs("section.key")
	assert.Error(t, err)

}
