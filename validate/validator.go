package validate

// Signatures validation.

import (
	"crypto"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

type Validator struct {
	backend *cache.RepoCache

	// FirstKey is the key used to sign the first commit.
	FirstKey *identity.Key

	// versions holds all the Identity Versions ordered by lamport time.
	versions       []*versionInfo
	// keyring holds all the current and past keys along with their expire time.
	keyring        openpgp.EntityList
	// keyCommit maps the key id to the commit which introduced that key.
	keyCommit      map[uint64]*object.Commit
	// checkedCommits holds the valid already-checked commits.
	checkedCommits map[repository.Hash]bool
}


// versionInfo contains details about a Version of an Identity, including
// the added and removed keys, if any.
type versionInfo struct {
	Version     *identity.Version
	Identity    *identity.Identity
	KeysAdded   []*identity.Key
	KeysRemoved []*identity.Key
	Commit      *object.Commit
}

type ByLamportTime []*versionInfo

func (a ByLamportTime) Len() int           { return len(a) }
func (a ByLamportTime) Less(i, j int) bool { return a[i].Version.Time() < a[j].Version.Time() }
func (a ByLamportTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }


// NewValidator creates a validator for the current identities snapshot.
// If identities are changed a new Validator instance should be used.
//
// The returned instance can be used to verify multiple git refs
// from the main repository against the keychain built from the
// identities snapshot loaded initially.
func NewValidator(backend *cache.RepoCache) (*Validator, error) {
	var err error

	v := &Validator{
		backend:        backend,
		keyring:        make(openpgp.EntityList, 0),
		keyCommit:      make(map[uint64]*object.Commit),
		checkedCommits: make(map[repository.Hash]bool),
	}

	v.versions, err = v.readVersionsInfo()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read identity versions")
	}

	sort.Sort(ByLamportTime(v.versions))

	if len(v.versions) > 0 {
		lastInfo := v.versions[0]
		for _, info := range v.versions[1:] {
			if len(info.KeysAdded) + len(info.KeysRemoved) > 0 && info.Version.Time() == lastInfo.Version.Time() {
				return nil, fmt.Errorf("multiple versions with the same lamport time: %d in commits %s %s", lastInfo.Version.Time(), lastInfo.Version.CommitHash(), info.Version.CommitHash())
			}
			lastInfo = info
		}
	}

	v.FirstKey, err = v.validateIdentities()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate identities")
	}

	return v, nil
}

// KeyCommitHash reports the hash of the commit associated with the Identity
// Version introducing the key using the specified keyId, if any.
func (v *Validator) KeyCommitHash(keyId uint64) string {
	commit := v.keyCommit[keyId]
	if commit == nil {
		return ""
	}
	return commit.Hash.String()
}

// readVersionsInfo stores all the operations ever done on each identity.
// Checks the keys introduced by the versions to be unique.
func (v *Validator) readVersionsInfo() ([]*versionInfo, error) {
	versions := make([]*versionInfo, 0)
	// Iterate the identity ids in a random order.
	for _, id := range v.backend.AllIdentityIds() {
		identityCache, err := v.backend.ResolveIdentity(id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve identity %s", id)
		}

		lastVersionKeys := make(map[uint64]*identity.Key)
		for _, version := range identityCache.Identity.Versions() {
			// Load the commit.
			hash := version.CommitHash()
			commit, err := v.backend.ResolveCommit(hash)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read commit %s for identity %s", hash, identityCache.Id())
			}

			versionKeys := make(map[uint64]*identity.Key)

			// Iterate the keys to see which one has been added in this version.
			keysAdded := make([]*identity.Key, 0)
			for _, key := range version.Keys() {
				pubkey := key.PublicKey()
				if _, present := lastVersionKeys[pubkey.KeyId]; present {
					// The key was already present in the previous version.
					delete(lastVersionKeys, pubkey.KeyId)
				} else {
					// The key was introduced in this version.
					keysAdded = append(keysAdded, key)
					if otherCommit, present := v.keyCommit[pubkey.KeyId]; present {
						// It's simpler to require keyIds to be unique than
						// to support non-unique keys.
						return nil, fmt.Errorf("keys with identical keyId introduced in commits %s and %s", otherCommit.Hash, commit.Hash)
					}
					v.keyCommit[pubkey.KeyId] = commit
				}
				versionKeys[pubkey.KeyId] = key
			}

			// The remaining keys have been removed.
			keysRemoved := make([]*identity.Key, 0, len(lastVersionKeys))
			for _, key := range lastVersionKeys {
				keysRemoved = append(keysRemoved, key)
			}

			versions = append(versions, &versionInfo{version, identityCache.Identity, keysAdded, keysRemoved, commit})

			lastVersionKeys = versionKeys
		}
	}
	return versions, nil
}

// validateIdentities checks the identity operations have been properly signed.
// Sets the key used to sign the first commit.
func (v *Validator) validateIdentities() (*identity.Key, error) {
	var firstKey *identity.Key

	// Iterate the ordered versions to check each of them.
	for _, info := range v.versions {
		if firstKey == nil {
			// For the first commit we update the keyring beforehand,
			// as it should be signed with the key it introduces.
			v.updateKeyring(info)
		}

		signingKey, err := v.ValidateCommit(info.Version.CommitHash())
		if err != nil {
			return nil, errors.Wrapf(err, "invalid identity %s (%s) commit %s", info.Identity.Id(), info.Identity.Email(), info.Version.CommitHash())
		}

		if firstKey == nil {
			for _, key := range info.Version.Keys() {
				if key.PublicKey().KeyId == signingKey.KeyId {
					firstKey = key
				}
			}
		} else {
			v.updateKeyring(info)
		}
	}

	return firstKey, nil
}

func (v *Validator) updateKeyring(info *versionInfo) {
	for _, key := range info.KeysRemoved {
		for _, entity := range v.keyring {
			pubkey := key.PublicKey()
			if entity.PrimaryKey.KeyId == pubkey.KeyId {
				lifetime := info.Commit.Committer.When.Sub(v.keyCommit[pubkey.KeyId].Committer.When)
				lifetimeSecs := uint32(lifetime.Seconds())
				// It's only one.
				for _, i := range entity.Identities {
					i.SelfSignature.KeyLifetimeSecs = &lifetimeSecs
				}
				break
			}
		}
	}
	for _, key := range info.KeysAdded {
		e := &openpgp.Entity{
			PrimaryKey: key.PublicKey(),
			Identities: make(map[string]*openpgp.Identity),
		}

		creationTime := info.Commit.Committer.When
		uid := packet.NewUserId(info.Identity.Name(), "", info.Identity.Email())
		isPrimaryId := true
		e.Identities[uid.Id] = &openpgp.Identity{
			Name:   uid.Id,
			UserId: uid,
			SelfSignature: &packet.Signature{
				CreationTime:    creationTime,
				SigType:         packet.SigTypePositiveCert,
				PubKeyAlgo:      packet.PubKeyAlgoRSA,
				Hash:            crypto.SHA256,
				KeyLifetimeSecs: nil,
				IsPrimaryId:     &isPrimaryId,
				FlagsValid:      true,
				FlagSign:        true,
				FlagCertify:     true,
				IssuerKeyId:     &e.PrimaryKey.KeyId,
			},
		}

		v.keyring = append(v.keyring, e)
	}
}

// ValidateCommit checks the commit signature along with the key's expire time,
// after checking all the parents recursively.
// Returns the pubkey used to sign the specified commit, or an error.
func (v *Validator) ValidateCommit(hash repository.Hash) (*packet.PublicKey, error) {
	if v.checkedCommits[hash] {
		return nil, nil
	}

	commit, err := v.backend.ResolveCommit(hash)
	if err != nil {
		return nil, err
	}

	for _, h := range commit.ParentHashes {
		_, err = v.ValidateCommit(repository.Hash(h.String()))
		if err != nil {
			return nil, err
		}
	}

	signingKey, err := v.verifyCommitSignature(commit)
	if err != nil {
		return nil, errors.Wrap(err, "invalid signature")
	}

	v.checkedCommits[hash] = true
	return signingKey, nil
}

// verifyCommitSignature returns which public key was able to verify the commit
// or an error.
func (v *Validator) verifyCommitSignature(commit *object.Commit) (*packet.PublicKey, error) {
	if commit.PGPSignature == "" {
		return nil, errors.New("commit is not signed")
	}

	signature, err := dearmorSignature(commit.PGPSignature)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dearmor PGP signature")
	}

	if signature.IssuerKeyId == nil {
		// We require this because otherwise it would be expensive to
		// iterate the keys to check which one can verify the signature.
		// openpgp.CheckDetachedSignature has the same expectation.
		return nil, errors.New("signature doesn't have an issuer")
	}

	// Encode commit components excluding the signature.
	// This is the content to be signed.
	encoded := &plumbing.MemoryObject{}
	if err := commit.EncodeWithoutSignature(encoded); err != nil {
		return nil, err
	}
	er, err := encoded.Reader()
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(er)

	key, err := v.searchKey(signature, body)
	if err != nil {
		return nil, err
	}

	// Check the committer email of the git commit matches
	// the email of the git-bug identity.
	var identity_ *openpgp.Identity
	emails := make([]string, len(key.Entity.Identities))
	i := 0
	for _, ei := range key.Entity.Identities {
		if ei.UserId.Email == commit.Committer.Email {
			identity_ = ei
			break
		}
		emails[i] = ei.UserId.Email
		i++
	}
	if identity_ == nil {
		return nil, fmt.Errorf("git commit committer-email does not match the identity-email: %s vs %s",
			commit.Committer.Email, strings.Join(emails, ","))
	}

	start := identity_.SelfSignature.CreationTime
	if start.After(commit.Committer.When) {
		return nil, fmt.Errorf("key used to sign commit was created after the commit %s", commit.Hash)
	}
	if identity_.SelfSignature.KeyLifetimeSecs != nil {
		expiry := start.Add(time.Duration(*identity_.SelfSignature.KeyLifetimeSecs))
		if expiry.Before(commit.Committer.When) {
			return nil, fmt.Errorf("key used to sign commit %s on %s expired on %s",
				commit.Hash, commit.Committer.When.Format(time.Stamp), expiry.Format(time.Stamp))
		}
	}

	return key.PublicKey, nil
}

// searchKey searches for a key which can verify the signature.
// It does not check the expire time.
func (v *Validator) searchKey(signature *packet.Signature, body []byte) (*openpgp.Key, error) {
	for _, key := range v.keyring.KeysById(*signature.IssuerKeyId) {
		signed := signature.Hash.New()
		_, err := signed.Write(body)
		if err != nil {
			return nil, err
		}
		err = key.PublicKey.VerifySignature(signed, signature)
		if err == nil {
			return &key, nil
		}
	}

	return nil, errors.New("no key can verify the signature")
}

// dearmorSignature decodes an armored signature.
func dearmorSignature(armoredSignature string) (*packet.Signature, error) {
	block, err := armor.Decode(strings.NewReader(armoredSignature))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dearmor signature")
	}
	reader := packet.NewReader(block.Body)
	p, err := reader.Next()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read signature packet")
	}
	sig, ok := p.(*packet.Signature)
	if !ok {
		// https://tools.ietf.org/html/rfc4880#section-5.2.3
		return nil, errors.New("failed to parse signature as Version 4 Signature Packet Format")
	}
	if sig == nil {
		// The optional "Issuer" field "(8-octet Key ID)" is missing.
		// https://tools.ietf.org/html/rfc4880#section-5.2.3.5
		return nil, fmt.Errorf("missing Issuer Key ID")
	}
	return sig, nil
}
