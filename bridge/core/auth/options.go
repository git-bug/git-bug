package auth

type listOptions struct {
	target string
	kind   map[CredentialKind]interface{}
	meta   map[string]string
}

type ListOption func(opts *listOptions)

func matcher(opts []ListOption) *listOptions {
	result := &listOptions{}
	for _, opt := range opts {
		opt(result)
	}
	return result
}

func (opts *listOptions) Match(cred Credential) bool {
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

func WithTarget(target string) ListOption {
	return func(opts *listOptions) {
		opts.target = target
	}
}

// WithKind match credentials with the given kind. Can be specified multiple times.
func WithKind(kind CredentialKind) ListOption {
	return func(opts *listOptions) {
		if opts.kind == nil {
			opts.kind = make(map[CredentialKind]interface{})
		}
		opts.kind[kind] = nil
	}
}

func WithMeta(key string, val string) ListOption {
	return func(opts *listOptions) {
		if opts.meta == nil {
			opts.meta = make(map[string]string)
		}
		opts.meta[key] = val
	}
}
