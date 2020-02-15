package auth

type options struct {
	target string
	kind   map[CredentialKind]interface{}
	meta   map[string]string
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

	_, has := opts.kind[cred.Kind()]
	if len(opts.kind) > 0 && !has {
		return false
	}

	for key, val := range opts.meta {
		if v, ok := cred.GetMetadata(key); !ok || v != val {
			return false
		}
	}

	return true
}

func WithTarget(target string) Option {
	return func(opts *options) {
		opts.target = target
	}
}

// WithKind match credentials with the given kind. Can be specified multiple times.
func WithKind(kind CredentialKind) Option {
	return func(opts *options) {
		if opts.kind == nil {
			opts.kind = make(map[CredentialKind]interface{})
		}
		opts.kind[kind] = nil
	}
}

func WithMeta(key string, val string) Option {
	return func(opts *options) {
		if opts.meta == nil {
			opts.meta = make(map[string]string)
		}
		opts.meta[key] = val
	}
}
