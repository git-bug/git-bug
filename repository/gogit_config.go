package repository

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/config"
)

var _ Config = &goGitConfig{}

type goGitConfig struct {
	repo *gogit.Repository
}

func newGoGitConfig(repo *gogit.Repository) *goGitConfig {
	return &goGitConfig{repo: repo}
}

func (ggc *goGitConfig) StoreString(key, value string) error {
	cfg, err := ggc.repo.Config()
	if err != nil {
		return err
	}

	split := strings.Split(key, ".")

	switch {
	case len(split) <= 1:
		return fmt.Errorf("invalid key")
	case len(split) == 2:
		cfg.Raw.Section(split[0]).SetOption(split[1], value)
	default:
		section := split[0]
		subsection := strings.Join(split[1:len(split)-2], ".")
		option := split[len(split)-1]
		cfg.Raw.Section(section).Subsection(subsection).SetOption(option, value)
	}

	return ggc.repo.SetConfig(cfg)
}

func (ggc *goGitConfig) StoreTimestamp(key string, value time.Time) error {
	return ggc.StoreString(key, strconv.Itoa(int(value.Unix())))
}

func (ggc *goGitConfig) StoreBool(key string, value bool) error {
	return ggc.StoreString(key, strconv.FormatBool(value))
}

func (ggc *goGitConfig) ReadAll(keyPrefix string) (map[string]string, error) {
	cfg, err := ggc.repo.Config()
	if err != nil {
		return nil, err
	}

	split := strings.Split(keyPrefix, ".")

	var opts config.Options

	switch {
	case len(split) < 1:
		return nil, fmt.Errorf("invalid key prefix")
	case len(split) == 1:
		opts = cfg.Raw.Section(split[0]).Options
	default:
		section := split[0]
		subsection := strings.Join(split[1:len(split)-1], ".")
		opts = cfg.Raw.Section(section).Subsection(subsection).Options
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("invalid section")
	}

	if keyPrefix[len(keyPrefix)-1:] != "." {
		keyPrefix += "."
	}

	result := make(map[string]string, len(opts))
	for _, opt := range opts {
		result[keyPrefix+opt.Key] = opt.Value
	}

	return result, nil
}

func (ggc *goGitConfig) ReadBool(key string) (bool, error) {
	val, err := ggc.ReadString(key)
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(val)
}

func (ggc *goGitConfig) ReadString(key string) (string, error) {
	cfg, err := ggc.repo.Config()
	if err != nil {
		return "", err
	}

	split := strings.Split(key, ".")

	if len(split) <= 1 {
		return "", fmt.Errorf("invalid key")
	}

	sectionName := split[0]
	if !cfg.Raw.HasSection(sectionName) {
		return "", ErrNoConfigEntry
	}
	section := cfg.Raw.Section(sectionName)

	switch {
	case len(split) == 2:
		optionName := split[1]
		if !section.HasOption(optionName) {
			return "", ErrNoConfigEntry
		}
		if len(section.OptionAll(optionName)) > 1 {
			return "", ErrMultipleConfigEntry
		}
		return section.Option(optionName), nil
	default:
		subsectionName := strings.Join(split[1:len(split)-2], ".")
		optionName := split[len(split)-1]
		if !section.HasSubsection(subsectionName) {
			return "", ErrNoConfigEntry
		}
		subsection := section.Subsection(subsectionName)
		if !subsection.HasOption(optionName) {
			return "", ErrNoConfigEntry
		}
		if len(subsection.OptionAll(optionName)) > 1 {
			return "", ErrMultipleConfigEntry
		}
		return subsection.Option(optionName), nil
	}
}

func (ggc *goGitConfig) ReadTimestamp(key string) (time.Time, error) {
	value, err := ggc.ReadString(key)
	if err != nil {
		return time.Time{}, err
	}
	return ParseTimestamp(value)
}

func (ggc *goGitConfig) RemoveAll(keyPrefix string) error {
	cfg, err := ggc.repo.Config()
	if err != nil {
		return err
	}

	split := strings.Split(keyPrefix, ".")

	switch {
	case len(split) < 1:
		return fmt.Errorf("invalid key prefix")
	case len(split) == 1:
		if len(cfg.Raw.Section(split[0]).Options) > 0 {
			cfg.Raw.RemoveSection(split[0])
		} else {
			return fmt.Errorf("invalid key prefix")
		}
	default:
		section := split[0]
		rest := strings.Join(split[1:], ".")

		if cfg.Raw.Section(section).HasSubsection(rest) {
			cfg.Raw.RemoveSubsection(section, rest)
		} else {
			if cfg.Raw.Section(section).HasOption(rest) {
				cfg.Raw.Section(section).RemoveOption(rest)
			} else {
				return fmt.Errorf("invalid key prefix")
			}
		}
	}

	return ggc.repo.SetConfig(cfg)
}
