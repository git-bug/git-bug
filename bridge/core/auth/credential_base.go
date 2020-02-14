package auth

import (
	"fmt"
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

func newCredentialBaseFromConfig(conf map[string]string) (*credentialBase, error) {
	base := &credentialBase{
		target: conf[configKeyTarget],
		meta:   metaFromConfig(conf),
	}

	if createTime, ok := conf[configKeyCreateTime]; ok {
		t, err := repository.ParseTimestamp(createTime)
		if err != nil {
			return nil, err
		}
		base.createTime = t
	} else {
		return nil, fmt.Errorf("missing create time")
	}

	salt, err := saltFromConfig(conf)
	if err != nil {
		return nil, err
	}
	base.salt = salt

	return base, nil
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
