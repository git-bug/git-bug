package repository

import (
	"os"
	"path/filepath"

	"github.com/99designs/keyring"
)

type Item = keyring.Item

var ErrKeyringKeyNotFound = keyring.ErrKeyNotFound

// Keyring provides the uniform interface over the underlying backends
type Keyring interface {
	// Returns an Item matching the key or ErrKeyringKeyNotFound
	Get(key string) (Item, error)
	// Stores an Item on the keyring
	Set(item Item) error
	// Removes the item with matching key
	Remove(key string) error
	// Provides a slice of all keys stored on the keyring
	Keys() ([]string, error)
}

func defaultKeyring() (Keyring, error) {
	ucd, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return keyring.Open(keyring.Config{
		// only use the file backend until https://github.com/99designs/keyring/issues/74 is resolved
		AllowedBackends: []keyring.BackendType{
			keyring.FileBackend,
		},

		ServiceName: "git-bug",

		// Fallback encrypted file
		FileDir: filepath.Join(ucd, "git-bug", "keyring"),
		// As we write the file in the user's config directory, this file should already be protected by the OS against
		// other user's access. We actually don't terribly need to protect it further and a password prompt across all
		// UI's would be a pain. Therefore we use here a constant password so the file will be unreadable by generic file
		// scanners if the user's machine get compromised.
		FilePasswordFunc: func(string) (string, error) {
			return "git-bug", nil
		},
	})
}

// replaceKeyring allow to replace the Keyring of the underlying repo
type replaceKeyring struct {
	TestedRepo
	keyring Keyring
}

func (rk replaceKeyring) Keyring() Keyring {
	return rk.keyring
}
