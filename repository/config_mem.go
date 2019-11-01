package repository

import (
	"strconv"
	"strings"
	"time"
)

var _ Config = &memConfig{}

type memConfig struct {
	config map[string]string
}

func newMemConfig(config map[string]string) *memConfig {
	return &memConfig{config: config}
}

func (mc *memConfig) StoreString(key, value string) error {
	mc.config[key] = value
	return nil
}

func (mc *memConfig) StoreBool(key string, value bool) error {
	return mc.StoreString(key, strconv.FormatBool(value))
}

func (mc *memConfig) StoreTimestamp(key string, value time.Time) error {
	return mc.StoreString(key, strconv.Itoa(int(value.Unix())))
}

func (mc *memConfig) ReadAll(keyPrefix string) (map[string]string, error) {
	result := make(map[string]string)
	for key, val := range mc.config {
		if strings.HasPrefix(key, keyPrefix) {
			result[key] = val
		}
	}
	return result, nil
}

func (mc *memConfig) ReadString(key string) (string, error) {
	// unlike git, the mock can only store one value for the same key
	val, ok := mc.config[key]
	if !ok {
		return "", ErrNoConfigEntry
	}

	return val, nil
}

func (mc *memConfig) ReadBool(key string) (bool, error) {
	// unlike git, the mock can only store one value for the same key
	val, ok := mc.config[key]
	if !ok {
		return false, ErrNoConfigEntry
	}

	return strconv.ParseBool(val)
}

func (mc *memConfig) ReadTimestamp(key string) (*time.Time, error) {
	value, err := mc.ReadString(key)
	if err != nil {
		return nil, err
	}
	timestamp, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}

	t := time.Unix(int64(timestamp), 0)
	return &t, nil
}

// RmConfigs remove all key/value pair matching the key prefix
func (mc *memConfig) RemoveAll(keyPrefix string) error {
	for key := range mc.config {
		if strings.HasPrefix(key, keyPrefix) {
			delete(mc.config, key)
		}
	}
	return nil
}
