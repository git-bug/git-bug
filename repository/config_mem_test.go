package repository

import "testing"

func TestNewMemConfig(t *testing.T) {
	testConfig(t, NewMemConfig())
}
