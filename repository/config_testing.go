package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testConfig(t *testing.T, config Config) {
	// string
	err := config.StoreString("section.key", "value")
	require.NoError(t, err)

	val, err := config.ReadString("section.key")
	require.NoError(t, err)
	require.Equal(t, "value", val)

	// bool
	err = config.StoreBool("section.true", true)
	require.NoError(t, err)

	val2, err := config.ReadBool("section.true")
	require.NoError(t, err)
	require.Equal(t, true, val2)

	// timestamp
	err = config.StoreTimestamp("section.time", time.Unix(1234, 0))
	require.NoError(t, err)

	val3, err := config.ReadTimestamp("section.time")
	require.NoError(t, err)
	require.Equal(t, time.Unix(1234, 0), val3)

	// ReadAll
	configs, err := config.ReadAll("section")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"section.key":  "value",
		"section.true": "true",
		"section.time": "1234",
	}, configs)

	// RemoveAll
	err = config.RemoveAll("section.true")
	require.NoError(t, err)

	configs, err = config.ReadAll("section")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"section.key":  "value",
		"section.time": "1234",
	}, configs)

	_, err = config.ReadBool("section.true")
	require.Equal(t, ErrNoConfigEntry, err)

	err = config.RemoveAll("section.nonexistingkey")
	require.Error(t, err)

	err = config.RemoveAll("section.key")
	require.NoError(t, err)

	_, err = config.ReadString("section.key")
	require.Equal(t, ErrNoConfigEntry, err)

	err = config.RemoveAll("nonexistingsection")
	require.Error(t, err)

	err = config.RemoveAll("section.time")
	require.NoError(t, err)

	err = config.RemoveAll("section")
	require.Error(t, err)

	_, err = config.ReadString("section.key")
	require.Error(t, err)

	err = config.RemoveAll("section.key")
	require.Error(t, err)

	// section + subsections
	require.NoError(t, config.StoreString("section.opt1", "foo"))
	require.NoError(t, config.StoreString("section.opt2", "foo2"))
	require.NoError(t, config.StoreString("section.subsection.opt1", "foo3"))
	require.NoError(t, config.StoreString("section.subsection.opt2", "foo4"))
	require.NoError(t, config.StoreString("section.subsection.subsection.opt1", "foo5"))
	require.NoError(t, config.StoreString("section.subsection.subsection.opt2", "foo6"))

	all, err := config.ReadAll("section")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"section.opt1":                       "foo",
		"section.opt2":                       "foo2",
		"section.subsection.opt1":            "foo3",
		"section.subsection.opt2":            "foo4",
		"section.subsection.subsection.opt1": "foo5",
		"section.subsection.subsection.opt2": "foo6",
	}, all)

	all, err = config.ReadAll("section.subsection")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"section.subsection.opt1":            "foo3",
		"section.subsection.opt2":            "foo4",
		"section.subsection.subsection.opt1": "foo5",
		"section.subsection.subsection.opt2": "foo6",
	}, all)

	all, err = config.ReadAll("section.subsection.subsection")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"section.subsection.subsection.opt1": "foo5",
		"section.subsection.subsection.opt2": "foo6",
	}, all)
}
