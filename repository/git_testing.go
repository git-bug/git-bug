package repository

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

// This is intended for testing only

func CreateTestRepo(bare bool) TestedRepo {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	var creator func(string) (*GitRepo, error)

	if bare {
		creator = InitBareGitRepo
	} else {
		creator = InitGitRepo
	}

	repo, err := creator(dir)
	if err != nil {
		log.Fatal(err)
	}

	config := repo.LocalConfig()
	if err := config.StoreString("user.name", "testuser"); err != nil {
		log.Fatal("failed to set user.name for test repository: ", err)
	}
	if err := config.StoreString("user.email", "testuser@example.com"); err != nil {
		log.Fatal("failed to set user.email for test repository: ", err)
	}

	return repo
}

// CreatePubkey returns an armored public PGP key.
func CreatePubkey(t *testing.T) string {
	// Generate a key pair for signing commits.
	pgpEntity, err := openpgp.NewEntity("First Last", "", "fl@example.org", nil)
	require.NoError(t, err)

	// Armor the public part.
	pubBuilder := &strings.Builder{}
	w, err := armor.Encode(pubBuilder, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	err = pgpEntity.Serialize(w)
	require.NoError(t, err)
	err = w.Close()
	require.NoError(t, err)
	armoredPub := pubBuilder.String()
	return armoredPub
}

// SetupSigningKey creates a GPG key and sets up the local config so it's used.
// The key id is set as "user.signingkey". For the key to be found, a `gpg`
// wrapper which uses only a custom keyring is created and set as "gpg.program".
// Finally "commit.gpgsign" is set to true so the signing takes place.
//
// Returns the armored public key.
func SetupSigningKey(t *testing.T, repo TestedRepo, email string) string {
	keyId, armoredPub, gpgWrapper := CreateKey(t, email)

	SetupKey(t, repo, email, keyId, gpgWrapper)

	return armoredPub
}

func SetupKey(t *testing.T, repo TestedRepo, email, keyId, gpgWrapper string) {
	config := repo.LocalConfig()

	if email != "" {
		err := config.StoreString("user.email", email)
		require.NoError(t, err)
	}

	if keyId != "" {
		err := config.StoreString("user.signingkey", keyId)
		require.NoError(t, err)
	}

	if gpgWrapper != "" {
		err := config.StoreString("gpg.program", gpgWrapper)
		require.NoError(t, err)
	}

	err := config.StoreString("commit.gpgsign", "true")
	require.NoError(t, err)
}

func CreateKey(t *testing.T, email string) (keyId, armoredPub, gpgWrapper string) {
	// Generate a key pair for signing commits.
	entity, err := openpgp.NewEntity("First Last", "", email, nil)
	require.NoError(t, err)

	keyId = entity.PrivateKey.KeyIdString()

	// Armor the private part.
	privBuilder := &strings.Builder{}
	w, err := armor.Encode(privBuilder, openpgp.PrivateKeyType, nil)
	require.NoError(t, err)
	err = entity.SerializePrivate(w, nil)
	require.NoError(t, err)
	err = w.Close()
	require.NoError(t, err)
	armoredPriv := privBuilder.String()

	// Armor the public part.
	pubBuilder := &strings.Builder{}
	w, err = armor.Encode(pubBuilder, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	err = entity.Serialize(w)
	require.NoError(t, err)
	err = w.Close()
	require.NoError(t, err)
	armoredPub = pubBuilder.String()

	// Create a custom gpg keyring to be used when creating commits with `git`.
	keyring, err := ioutil.TempFile("", "keyring")
	require.NoError(t, err)

	// Import the armored private key to the custom keyring.
	priv, err := ioutil.TempFile("", "privkey")
	require.NoError(t, err)
	_, err = fmt.Fprint(priv, armoredPriv)
	require.NoError(t, err)
	err = priv.Close()
	require.NoError(t, err)
	err = exec.Command("gpg", "--no-default-keyring", "--keyring", keyring.Name(), "--import", priv.Name()).Run()
	require.NoError(t, err)

	// Import the armored public key to the custom keyring.
	pub, err := ioutil.TempFile("", "pubkey")
	require.NoError(t, err)
	_, err = fmt.Fprint(pub, armoredPub)
	require.NoError(t, err)
	err = pub.Close()
	require.NoError(t, err)
	err = exec.Command("gpg", "--no-default-keyring", "--keyring", keyring.Name(), "--import", pub.Name()).Run()
	require.NoError(t, err)

	// Use a gpg wrapper to use a custom keyring containing GPGKeyID.
	gpgWrapper = createGPGWrapper(t, keyring.Name())

	return
}

// createGPGWrapper creates a shell script running gpg with a specific keyring.
func createGPGWrapper(t *testing.T, keyringPath string) string {
	file, err := ioutil.TempFile("", "gpgwrapper")
	require.NoError(t, err)

	_, err = fmt.Fprintf(file, `#!/bin/sh
exec gpg --no-default-keyring --keyring="%s" "$@"
`, keyringPath)
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	err = os.Chmod(file.Name(), os.FileMode(0700))
	require.NoError(t, err)

	return file.Name()
}

func SetupReposAndRemote() (repoA, repoB, remote TestedRepo) {
	repoA = CreateTestRepo(false)
	repoB = CreateTestRepo(false)
	remote = CreateTestRepo(true)

	remoteAddr := "file://" + remote.GetPath()

	err := repoA.AddRemote("origin", remoteAddr)
	if err != nil {
		log.Fatal(err)
	}

	err = repoB.AddRemote("origin", remoteAddr)
	if err != nil {
		log.Fatal(err)
	}

	return repoA, repoB, remote
}
