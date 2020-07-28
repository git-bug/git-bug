package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/repository"
)

type credentialBase struct {
	target     string
	createTime time.Time
	salt       []byte
	meta       map[string]string
}

func newCredentialBase(target string) *credentialBase {
	return &credentialBase{
		target:     target,
		createTime: time.Now(),
		salt:       makeSalt(),
	}
}

func makeSalt() []byte {
	result := make([]byte, 16)
	_, err := rand.Read(result)
	if err != nil {
		panic(err)
	}
	return result
}

func newCredentialBaseFromData(data map[string]string) (*credentialBase, error) {
	base := &credentialBase{
		target: data[keyringKeyTarget],
		meta:   metaFromData(data),
	}

	if createTime, ok := data[keyringKeyCreateTime]; ok {
		t, err := repository.ParseTimestamp(createTime)
		if err != nil {
			return nil, err
		}
		base.createTime = t
	} else {
		return nil, fmt.Errorf("missing create time")
	}

	salt, err := saltFromData(data)
	if err != nil {
		return nil, err
	}
	base.salt = salt

	return base, nil
}

func metaFromData(data map[string]string) map[string]string {
	result := make(map[string]string)
	for key, val := range data {
		if strings.HasPrefix(key, keyringKeyPrefixMeta) {
			key = strings.TrimPrefix(key, keyringKeyPrefixMeta)
			result[key] = val
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func saltFromData(data map[string]string) ([]byte, error) {
	val, ok := data[keyringKeySalt]
	if !ok {
		return nil, fmt.Errorf("no credential salt found")
	}
	return base64.StdEncoding.DecodeString(val)
}

func (cb *credentialBase) Target() string {
	return cb.target
}

func (cb *credentialBase) CreateTime() time.Time {
	return cb.createTime
}

func (cb *credentialBase) Salt() []byte {
	return cb.salt
}

func (cb *credentialBase) validate() error {
	if cb.target == "" {
		return fmt.Errorf("missing target")
	}
	if cb.createTime.IsZero() || cb.createTime.Equal(time.Time{}) {
		return fmt.Errorf("missing creation time")
	}
	if !core.TargetExist(cb.target) {
		return fmt.Errorf("unknown target")
	}
	return nil
}

func (cb *credentialBase) Metadata() map[string]string {
	return cb.meta
}

func (cb *credentialBase) GetMetadata(key string) (string, bool) {
	val, ok := cb.meta[key]
	return val, ok
}

func (cb *credentialBase) SetMetadata(key string, value string) {
	if cb.meta == nil {
		cb.meta = make(map[string]string)
	}
	cb.meta[key] = value
}
