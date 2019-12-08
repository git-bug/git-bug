package auth

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
)

type options struct {
	target string
	userId entity.Id
	kind   CredentialKind
}

type Option func(opts *options)

func matcher(opts []Option) *options {
	result := &options{}
	for _, opt := range opts {
		opt(result)
	}
	return result
}

func (opts *options) Match(cred Credential) bool {
	if opts.target != "" && cred.Target() != opts.target {
		return false
	}

	if opts.userId != "" && cred.UserId() != opts.userId {
		return false
	}

	if opts.kind != "" && cred.Kind() != opts.kind {
		return false
	}

	return true
}

func WithTarget(target string) Option {
	return func(opts *options) {
		opts.target = target
	}
}

func WithUser(user identity.Interface) Option {
	return func(opts *options) {
		opts.userId = user.Id()
	}
}

func WithUserId(userId entity.Id) Option {
	return func(opts *options) {
		opts.userId = userId
	}
}

func WithKind(kind CredentialKind) Option {
	return func(opts *options) {
		opts.kind = kind
	}
}
