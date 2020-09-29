package repository

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
)

var _ Config = &gitConfig{}

type gitConfig struct {
	cli          gitCli
	localityFlag string
}

func newGitConfig(cli gitCli, global bool) *gitConfig {
	localityFlag := "--local"
	if global {
		localityFlag = "--global"
	}
	return &gitConfig{
		cli:          cli,
		localityFlag: localityFlag,
	}
}

// StoreString store a single key/value pair in the config of the repo
func (gc *gitConfig) StoreString(key string, value string) error {
	_, err := gc.cli.runGitCommand("config", gc.localityFlag, "--replace-all", key, value)
	return err
}

func (gc *gitConfig) StoreBool(key string, value bool) error {
	return gc.StoreString(key, strconv.FormatBool(value))
}

func (gc *gitConfig) StoreTimestamp(key string, value time.Time) error {
	return gc.StoreString(key, strconv.Itoa(int(value.Unix())))
}

// ReadAll read all key/value pair matching the key prefix
func (gc *gitConfig) ReadAll(keyPrefix string) (map[string]string, error) {
	stdout, err := gc.cli.runGitCommand("config", gc.localityFlag, "--includes", "--get-regexp", keyPrefix)

	//   / \
	//  / ! \
	// -------
	//
	// There can be a legitimate error here, but I see no portable way to
	// distinguish them from the git error that say "no matching value exist"
	if err != nil {
		return nil, nil
	}

	lines := strings.Split(stdout, "\n")

	result := make(map[string]string, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		result[parts[0]] = parts[1]
	}

	return result, nil
}

func (gc *gitConfig) ReadString(key string) (string, error) {
	stdout, err := gc.cli.runGitCommand("config", gc.localityFlag, "--includes", "--get-all", key)

	//   / \
	//  / ! \
	// -------
	//
	// There can be a legitimate error here, but I see no portable way to
	// distinguish them from the git error that say "no matching value exist"
	if err != nil {
		return "", ErrNoConfigEntry
	}

	lines := strings.Split(stdout, "\n")

	if len(lines) == 0 {
		return "", ErrNoConfigEntry
	}
	if len(lines) > 1 {
		return "", ErrMultipleConfigEntry
	}

	return lines[0], nil
}

func (gc *gitConfig) ReadBool(key string) (bool, error) {
	val, err := gc.ReadString(key)
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(val)
}

func (gc *gitConfig) ReadTimestamp(key string) (time.Time, error) {
	value, err := gc.ReadString(key)
	if err != nil {
		return time.Time{}, err
	}
	return ParseTimestamp(value)
}

func (gc *gitConfig) rmSection(keyPrefix string) error {
	_, err := gc.cli.runGitCommand("config", gc.localityFlag, "--remove-section", keyPrefix)
	return err
}

func (gc *gitConfig) unsetAll(keyPrefix string) error {
	_, err := gc.cli.runGitCommand("config", gc.localityFlag, "--unset-all", keyPrefix)
	return err
}

// return keyPrefix section
// example: sectionFromKey(a.b.c.d) return a.b.c
func sectionFromKey(keyPrefix string) string {
	s := strings.Split(keyPrefix, ".")
	if len(s) == 1 {
		return keyPrefix
	}

	return strings.Join(s[:len(s)-1], ".")
}

// rmConfigs with git version lesser than 2.18
func (gc *gitConfig) rmConfigsGitVersionLT218(keyPrefix string) error {
	// try to remove key/value pair by key
	err := gc.unsetAll(keyPrefix)
	if err != nil {
		return gc.rmSection(keyPrefix)
	}

	m, err := gc.ReadAll(sectionFromKey(keyPrefix))
	if err != nil {
		return err
	}

	// if section doesn't have any left key/value remove the section
	if len(m) == 0 {
		return gc.rmSection(sectionFromKey(keyPrefix))
	}

	return nil
}

// RmConfigs remove all key/value pair matching the key prefix
func (gc *gitConfig) RemoveAll(keyPrefix string) error {
	// starting from git 2.18.0 sections are automatically deleted when the last existing
	// key/value is removed. Before 2.18.0 we should remove the section
	// see https://github.com/git/git/blob/master/Documentation/RelNotes/2.18.0.txt#L379
	lt218, err := gc.gitVersionLT218()
	if err != nil {
		return errors.Wrap(err, "getting git version")
	}

	if lt218 {
		return gc.rmConfigsGitVersionLT218(keyPrefix)
	}

	err = gc.unsetAll(keyPrefix)
	if err != nil {
		return gc.rmSection(keyPrefix)
	}

	return nil
}

func (gc *gitConfig) gitVersion() (*semver.Version, error) {
	versionOut, err := gc.cli.runGitCommand("version")
	if err != nil {
		return nil, err
	}
	return parseGitVersion(versionOut)
}

func parseGitVersion(versionOut string) (*semver.Version, error) {
	// extract the version and truncate potential bad parts
	// ex: 2.23.0.rc1 instead of 2.23.0-rc1
	r := regexp.MustCompile(`(\d+\.){1,2}\d+`)

	extracted := r.FindString(versionOut)
	if extracted == "" {
		return nil, fmt.Errorf("unreadable git version %s", versionOut)
	}

	version, err := semver.Make(extracted)
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func (gc *gitConfig) gitVersionLT218() (bool, error) {
	version, err := gc.gitVersion()
	if err != nil {
		return false, err
	}

	version218string := "2.18.0"
	gitVersion218, err := semver.Make(version218string)
	if err != nil {
		return false, err
	}

	return version.LT(gitVersion218), nil
}
