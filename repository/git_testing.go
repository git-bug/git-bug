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

// setupSigningKey creates a GPG key and sets up the local config so it's used.
// The key id is set as "user.signingkey". For the key to be found, a `gpg`
// wrapper which uses only a custom keyring is created and set as "gpg.program".
// Finally "commit.gpgsign" is set to true so the signing takes place.
func setupSigningKey(t *testing.T, repo TestedRepo) {
	config := repo.LocalConfig()

	// Generate a key pair for signing commits.
	entity, err := openpgp.NewEntity("First Last", "", "fl@example.org", nil)
	require.NoError(t, err)

	err = config.StoreString("user.signingkey", entity.PrivateKey.KeyIdString())
	require.NoError(t, err)

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
	armoredPub := pubBuilder.String()

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
	gpgWrapper := createGPGWrapper(t, keyring.Name())
	if err := config.StoreString("gpg.program", gpgWrapper); err != nil {
		log.Fatal("failed to set gpg.program for test repository: ", err)
	}

	if err := config.StoreString("commit.gpgsign", "true"); err != nil {
		log.Fatal("failed to set commit.gpgsign for test repository: ", err)
	}
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
