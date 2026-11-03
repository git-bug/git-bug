package repository

import (
	"os"
	"strconv"
	"time"

	"github.com/git-bug/gitconfig"
	"golang.org/x/sys/execabs"
)

// Config gives access to the git configuration of a repository, as `git config`
// does: reads see every scope merged, writes go to the repository's config file.
type Config interface {
	// Read reads the effective configuration as it is now: every scope merged
	// as git does, the last value of a key winning. The result is an immutable
	// snapshot: later updates don't change it.
	Read() (*gitconfig.Config, error)

	// Update edits the repository's config file with fn, atomically: if fn
	// returns an error, nothing is written, otherwise all its edits are.
	// Concurrent updates, from git or git-bug, never lose each other's edits.
	// fn must not keep the File after it returns.
	Update(fn func(f *gitconfig.File) error) error
}

func ParseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	timestamp, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	if timestamp > 1e14 {
		return time.Unix(0, timestamp), nil
	}
	return time.Unix(timestamp, 0), nil
}

var _ RepoConfig = &repoConfig{}

type repoConfig struct {
	config Config
}

// NewRepoConfig returns the RepoConfig of config.
func NewRepoConfig(config Config) RepoConfig {
	return &repoConfig{config: config}
}

// Config give access to the git configuration
func (r *repoConfig) Config() Config {
	return r.config
}

// GetUserName returns the name the user has used to configure git, "" if unset
func (r *repoConfig) GetUserName() (string, error) {
	cfg, err := r.config.Read()
	if err != nil {
		return "", err
	}
	return cfg.Value("user.name", ""), nil
}

// GetUserEmail returns the email address that the user has used to configure git, "" if unset
func (r *repoConfig) GetUserEmail() (string, error) {
	cfg, err := r.config.Read()
	if err != nil {
		return "", err
	}
	return cfg.Value("user.email", ""), nil
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (r *repoConfig) GetCoreEditor() (string, error) {
	// See https://git-scm.com/docs/git-var
	// The order of preference is the $GIT_EDITOR environment variable, then core.editor configuration, then $VISUAL, then $EDITOR, and then the default chosen at compile time, which is usually vi.

	if val, ok := os.LookupEnv("GIT_EDITOR"); ok {
		return val, nil
	}

	cfg, err := r.config.Read()
	if err != nil {
		return "", err
	}
	if val := cfg.Value("core.editor", ""); val != "" {
		return val, nil
	}

	if val, ok := os.LookupEnv("VISUAL"); ok {
		return val, nil
	}

	if val, ok := os.LookupEnv("EDITOR"); ok {
		return val, nil
	}

	priorities := []string{
		"editor",
		"nano",
		"vim",
		"vi",
		"emacs",
	}

	for _, cmd := range priorities {
		if _, err = execabs.LookPath(cmd); err == nil {
			return cmd, nil
		}

	}

	return "ed", nil
}

// GetRemotes returns the configured remotes repositories, with the first url
// of each.
func (r *repoConfig) GetRemotes() (map[string]string, error) {
	cfg, err := r.config.Read()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, name := range cfg.Subsections("remote") {
		urls := cfg.GetAll("remote." + name + ".url")
		if len(urls) > 0 {
			result[name] = urls[0].Value
		}
	}

	return result, nil
}
