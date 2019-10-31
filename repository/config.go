package repository

import "time"

// Config represent the common function interacting with the repository config storage
type Config interface {
	// Store writes a single key/value pair in the config of the repo
	Store(key string, value string) error

	// ReadAll reads all key/value pair matching the key prefix
	ReadAll(keyPrefix string) (map[string]string, error)

	// ReadBool read a single boolean value from the config
	// Return ErrNoConfigEntry or ErrMultipleConfigEntry if
	// there is zero or more than one entry for this key
	ReadBool(key string) (bool, error)

	// ReadBool read a single string value from the config
	// Return ErrNoConfigEntry or ErrMultipleConfigEntry if
	// there is zero or more than one entry for this key
	ReadString(key string) (string, error)

	// ReadTimestamp read a single timestamp value from the config
	// Return ErrNoConfigEntry or ErrMultipleConfigEntry if
	// there is zero or more than one entry for this key
	ReadTimestamp(key string) (*time.Time, error)

	// RemoveAll removes all key/value pair matching the key prefix
	RemoveAll(keyPrefix string) error
}
