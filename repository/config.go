package repository

import (
	"errors"
	"strconv"
	"time"
)

var (
	ErrNoConfigEntry       = errors.New("no config entry for the given key")
	ErrMultipleConfigEntry = errors.New("multiple config entry for the given key")
)

// Config represent the common function interacting with the repository config storage
type Config interface {
	ConfigRead
	ConfigWrite
}

type ConfigRead interface {
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
	ReadTimestamp(key string) (time.Time, error)
}

type ConfigWrite interface {
	// Store writes a single key/value pair in the config
	StoreString(key, value string) error

	// Store writes a key and timestamp value to the config
	StoreTimestamp(key string, value time.Time) error

	// Store writes a key and boolean value to the config
	StoreBool(key string, value bool) error

	// RemoveAll removes all key/value pair matching the key prefix
	RemoveAll(keyPrefix string) error
}

func ParseTimestamp(s string) (time.Time, error) {
	timestamp, err := strconv.Atoi(s)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(int64(timestamp), 0), nil
}

// mergeConfig is a helper to easily support RepoConfig.AnyConfig()
// from two separate local and global Config
func mergeConfig(local ConfigRead, global ConfigRead) *mergedConfig {
	return &mergedConfig{
		local:  local,
		global: global,
	}
}

var _ ConfigRead = &mergedConfig{}

type mergedConfig struct {
	local  ConfigRead
	global ConfigRead
}

func (m *mergedConfig) ReadAll(keyPrefix string) (map[string]string, error) {
	values, err := m.global.ReadAll(keyPrefix)
	if err != nil {
		return nil, err
	}
	locals, err := m.local.ReadAll(keyPrefix)
	if err != nil {
		return nil, err
	}
	for k, val := range locals {
		values[k] = val
	}
	return values, nil
}

func (m *mergedConfig) ReadBool(key string) (bool, error) {
	v, err := m.local.ReadBool(key)
	if err == nil {
		return v, nil
	}
	if err != ErrNoConfigEntry && err != ErrMultipleConfigEntry {
		return false, err
	}
	return m.global.ReadBool(key)
}

func (m *mergedConfig) ReadString(key string) (string, error) {
	val, err := m.local.ReadString(key)
	if err == nil {
		return val, nil
	}
	if err != ErrNoConfigEntry && err != ErrMultipleConfigEntry {
		return "", err
	}
	return m.global.ReadString(key)
}

func (m *mergedConfig) ReadTimestamp(key string) (time.Time, error) {
	val, err := m.local.ReadTimestamp(key)
	if err == nil {
		return val, nil
	}
	if err != ErrNoConfigEntry && err != ErrMultipleConfigEntry {
		return time.Time{}, err
	}
	return m.global.ReadTimestamp(key)
}
