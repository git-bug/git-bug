package repository

import (
	"strconv"
	"strings"
)

type runtimeConfig struct {
	config map[string]string
}

func newRuntimeConfig(config map[string]string) *runtimeConfig {
	return &runtimeConfig{config: config}
}

func (rtc *runtimeConfig) Store(key, value string) error {
	rtc.config[key] = value
	return nil
}

func (rtc *runtimeConfig) ReadAll(keyPrefix string) (map[string]string, error) {
	result := make(map[string]string)
	for key, val := range rtc.config {
		if strings.HasPrefix(key, keyPrefix) {
			result[key] = val
		}
	}
	return result, nil
}

func (rtc *runtimeConfig) ReadString(key string) (string, error) {
	// unlike git, the mock can only store one value for the same key
	val, ok := rtc.config[key]
	if !ok {
		return "", ErrNoConfigEntry
	}

	return val, nil
}

func (rtc *runtimeConfig) ReadBool(key string) (bool, error) {
	// unlike git, the mock can only store one value for the same key
	val, ok := rtc.config[key]
	if !ok {
		return false, ErrNoConfigEntry
	}

	return strconv.ParseBool(val)
}

// RmConfigs remove all key/value pair matching the key prefix
func (rtc *runtimeConfig) RemoveAll(keyPrefix string) error {
	for key := range rtc.config {
		if strings.HasPrefix(key, keyPrefix) {
			delete(rtc.config, key)
		}
	}
	return nil
}
