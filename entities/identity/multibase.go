package identity

import (
	"fmt"

	mbase "github.com/multiformats/go-multibase"
	"github.com/multiformats/go-varint"
)

// Multicodec algorithm codes for publicKeyMultibase encoding.
// Values from the multicodec table: https://github.com/multiformats/multicodec/blob/master/table.csv
// Picked from github.com/MetaMask/go-did-it@v1.0.0-pre1.
const (
	multibaseCodeEd25519 = uint64(0xed)   // Ed25519 public key (32 raw bytes)
	multibaseCodeP256    = uint64(0x1200) // P-256 public key (compressed point)
	multibaseCodeP384    = uint64(0x1201) // P-384 public key (compressed point)
	multibaseCodeP521    = uint64(0x1202) // P-521 public key (compressed point)
	multibaseCodeRSA     = uint64(0x1205) // RSA public key (PKIX DER)
)

// pubkeyMultibaseEncode encodes raw key bytes into a W3C DID publicKeyMultibase string:
// 'z' prefix (base58btc) + varint(code) + keyBytes.
func pubkeyMultibaseEncode(code uint64, keyBytes []byte) string {
	payload := append(varint.ToUvarint(code), keyBytes...)
	s, _ := mbase.Encode(mbase.Base58BTC, payload)
	return s
}

// pubkeyMultibaseDecode decodes a publicKeyMultibase string into its algorithm code and
// raw key bytes.
func pubkeyMultibaseDecode(mb string) (uint64, []byte, error) {
	enc, payload, err := mbase.Decode(mb)
	if err != nil {
		return 0, nil, err
	}
	if enc != mbase.Base58BTC {
		return 0, nil, fmt.Errorf("publicKeyMultibase: expected base58btc (z-prefix) encoding")
	}
	code, n, err := varint.FromUvarint(payload)
	if err != nil {
		return 0, nil, err
	}
	return code, payload[n:], nil
}
