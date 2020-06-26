package repository

import (
	"fmt"
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
	val, ok := mc.config[key]
	if !ok {
		return "", ErrNoConfigEntry
	}

	return val, nil
}

func (mc *MemConfig) ReadBool(key string) (bool, error) {
	// unlike git, the mock can only store one value for the same key
	val, ok := mc.config[key]
	if !ok {
		return false, ErrNoConfigEntry
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

// RmConfigs remove all key/value pair matching the key prefix
func (mc *MemConfig) RemoveAll(keyPrefix string) error {
	found := false
	for key := range mc.config {
		if strings.HasPrefix(key, keyPrefix) {
			delete(mc.config, key)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("section not found")
	}

	return nil
}
