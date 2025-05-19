package repository

import (
	"strconv"
	"strings"
	"time"
)

var _ Config = &MemConfig{}

type MemConfig struct {
	config map[string]string
}

func NewMemConfig() *MemConfig {
	return &MemConfig{
		config: make(map[string]string),
	}
}

func (mc *MemConfig) StoreString(key, value string) error {
	key = normalizeKey(key)
	mc.config[key] = value
	return nil
}

func (mc *MemConfig) StoreBool(key string, value bool) error {
	return mc.StoreString(key, strconv.FormatBool(value))
}

func (mc *MemConfig) StoreTimestamp(key string, value time.Time) error {
	return mc.StoreString(key, strconv.Itoa(int(value.Unix())))
}

func (mc *MemConfig) ReadAll(keyPrefix string) (map[string]string, error) {
	keyPrefix = normalizeKey(keyPrefix)
	result := make(map[string]string)
	for key, val := range mc.config {
		if strings.HasPrefix(key, keyPrefix) {
			result[key] = val
		}
	}
	return result, nil
}

func (mc *MemConfig) ReadString(key string) (string, error) {
	// unlike git, the mock can only store one value for the same key
	key = normalizeKey(key)
	val, ok := mc.config[key]
	if !ok {
		return "", newErrNoConfigEntry(key)
	}

	return val, nil
}

func (mc *MemConfig) ReadBool(key string) (bool, error) {
	// unlike git, the mock can only store one value for the same key
	val, err := mc.ReadString(key)
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(val)
}

func (mc *MemConfig) ReadTimestamp(key string) (time.Time, error) {
	value, err := mc.ReadString(key)
	if err != nil {
		return time.Time{}, err
	}

	timestamp, err := strconv.Atoi(value)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(int64(timestamp), 0), nil
}

// RemoveAll remove all key/value pair matching the key prefix
func (mc *MemConfig) RemoveAll(keyPrefix string) error {
	keyPrefix = normalizeKey(keyPrefix)
	for key := range mc.config {
		if strings.HasPrefix(key, keyPrefix) {
			delete(mc.config, key)
		}
	}

	return nil
}

func normalizeKey(key string) string {
	// this feels so wrong, but that's apparently how git behave.
	// only section and final segment are case insensitive, subsection in between are not.
	s := strings.Split(key, ".")
	s[0] = strings.ToLower(s[0])
	s[len(s)-1] = strings.ToLower(s[len(s)-1])
	return strings.Join(s, ".")
}
