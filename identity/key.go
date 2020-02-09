package identity

type Key struct {
	// The GPG fingerprint of the key
	Fingerprint string `json:"fingerprint"`
	PubKey      string `json:"pub_key"`
}

func (k *Key) Validate() error {
	// Todo

	return nil
}

func (k *Key) Clone() *Key {
	clone := *k
	return &clone
}
