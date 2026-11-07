package repository

import (
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/errors"
)

// deArmorSignature convert an armored (text serialized) signature into raw binary
func deArmorSignature(armoredSig io.Reader) (io.Reader, error) {
	block, err := armor.Decode(armoredSig)
	if err != nil {
		return nil, err
	}
	if block.Type != openpgp.SignatureType {
		return nil, errors.InvalidArgumentError("expected '" + openpgp.SignatureType + "', got: " + block.Type)
	}
	return block.Body, nil
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Refs are stored as git references, grouped by namespace:
//   - refs/<namespace>/<key> for the local refs
//   - refs/remotes/<remote>/<namespace>/<key> for the tracking refs of a remote

func refPrefix(namespace string) string {
	return "refs/" + namespace + "/"
}

func trackingRefPrefix(remote string, namespace string) string {
	return "refs/remotes/" + remote + "/" + namespace + "/"
}
