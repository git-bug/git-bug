package repository

import (
	"os"
	"path"

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

	backends := []keyring.BackendType{
		keyring.WinCredBackend,
		keyring.KeychainBackend,
		keyring.PassBackend,
		keyring.FileBackend,
	}

	return keyring.Open(keyring.Config{
		// TODO: ideally this would not be there, it disable the freedesktop backend on linux
		// due to https://github.com/99designs/keyring/issues/44
		AllowedBackends: backends,

		ServiceName: "git-bug",

		// MacOS keychain
		KeychainName:             "git-bug",
		KeychainTrustApplication: true,

		// KDE Wallet
		KWalletAppID:  "git-bug",
		KWalletFolder: "git-bug",

		// Windows
		WinCredPrefix: "git-bug",

		// freedesktop.org's Secret Service
		LibSecretCollectionName: "git-bug",

		// Pass (https://www.passwordstore.org/)
		PassPrefix: "git-bug",

		// Fallback encrypted file
		FileDir: path.Join(ucd, "git-bug", "keyring"),
		// As we write the file in the user's config directory, this file should already be protected by the OS against
		// other user's access. We actually don't terribly need to protect it further and a password prompt across all
		// UI's would be a pain. Therefore we use here a constant password so the file will be unreadable by generic file
		// scanners if the user's machine get compromised.
		FilePasswordFunc: func(string) (string, error) {
			return "git-bug", nil
		},
	})
}
