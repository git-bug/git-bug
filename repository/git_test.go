// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	repo := CreateTestRepo(false)
	defer CleanupTestRepos(t, repo)

	config := repo.LocalConfig()

	err := config.StoreString("section.key", "value")
	assert.NoError(t, err)

	val, err := config.ReadString("section.key")
	assert.Equal(t, "value", val)

	err = config.StoreString("section.true", "true")
	assert.NoError(t, err)

	val2, err := config.ReadBool("section.true")
	assert.Equal(t, true, val2)

	configs, err := config.ReadAll("section")
	assert.NoError(t, err)
	assert.Equal(t, configs, map[string]string{
		"section.key":  "value",
		"section.true": "true",
	})

	err = config.RemoveAll("section.true")
	assert.NoError(t, err)

	configs, err = config.ReadAll("section")
	assert.NoError(t, err)
	assert.Equal(t, configs, map[string]string{
		"section.key": "value",
	})

	_, err = config.ReadBool("section.true")
	assert.Equal(t, ErrNoConfigEntry, err)

	err = config.RemoveAll("section.nonexistingkey")
	assert.Error(t, err)

	err = config.RemoveAll("section.key")
	assert.NoError(t, err)

	_, err = config.ReadString("section.key")
	assert.Equal(t, ErrNoConfigEntry, err)

	err = config.RemoveAll("nonexistingsection")
	assert.Error(t, err)

	err = config.RemoveAll("section")
	assert.Error(t, err)

	_, err = config.ReadString("section.key")
	assert.Error(t, err)

	err = config.RemoveAll("section.key")
	assert.Error(t, err)
}
