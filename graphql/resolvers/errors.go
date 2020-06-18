package resolvers

import "errors"

// ErrNotAuthenticated is returned to the client if the user requests an action requiring authentication, and they are not authenticated.
var ErrNotAuthenticated = errors.New("not authenticated or read-only")
