package repository

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

var _ Config = &goGitConfig{}

type goGitConfig struct {
	ConfigRead
	ConfigWrite
}

func newGoGitLocalConfig(repo *gogit.Repository) *goGitConfig {
	return &goGitConfig{
		ConfigRead:  &goGitConfigReader{getConfig: repo.Config},
		ConfigWrite: &goGitConfigWriter{repo: repo},
	}
}

func newGoGitGlobalConfig() *goGitConfig {
	// TODO: replace that with go-git native implementation once it's supported
	// see: https://github.com/go-git/go-git
	// see: https://github.com/src-d/go-git/issues/760

	return &goGitConfig{
		ConfigRead: &goGitConfigReader{getConfig: func() (*config.Config, error) {
			return config.LoadConfig(config.GlobalScope)
		}},
		ConfigWrite: &configPanicWriter{},
	}
}

var _ ConfigRead = &goGitConfigReader{}

type goGitConfigReader struct {
	getConfig func() (*config.Config, error)
}

func (cr *goGitConfigReader) ReadAll(keyPrefix string) (map[string]string, error) {
	cfg, err := cr.getConfig()
	if err != nil {
		return nil, err
	}

	split := strings.Split(keyPrefix, ".")
	result := make(map[string]string)

	switch {
	case keyPrefix == "":
		for _, section := range cfg.Raw.Sections {
			for _, option := range section.Options {
				result[fmt.Sprintf("%s.%s", section.Name, option.Key)] = option.Value
			}
			for _, subsection := range section.Subsections {
				for _, option := range subsection.Options {
					result[fmt.Sprintf("%s.%s.%s", section.Name, subsection.Name, option.Key)] = option.Value
				}
			}
		}
	case len(split) == 1:
		if !cfg.Raw.HasSection(split[0]) {
			return nil, nil
		}
		section := cfg.Raw.Section(split[0])
		for _, option := range section.Options {
			result[fmt.Sprintf("%s.%s", section.Name, option.Key)] = option.Value
		}
		for _, subsection := range section.Subsections {
			for _, option := range subsection.Options {
				result[fmt.Sprintf("%s.%s.%s", section.Name, subsection.Name, option.Key)] = option.Value
			}
		}
	default:
		if !cfg.Raw.HasSection(split[0]) {
			return nil, nil
		}
		section := cfg.Raw.Section(split[0])
		rest := strings.Join(split[1:], ".")
		rest = strings.TrimSuffix(rest, ".")
		for _, subsection := range section.Subsections {
			if strings.HasPrefix(subsection.Name, rest) {
				for _, option := range subsection.Options {
					result[fmt.Sprintf("%s.%s.%s", section.Name, subsection.Name, option.Key)] = option.Value
				}
			}
		}
	}

	return result, nil
}

func (cr *goGitConfigReader) ReadBool(key string) (bool, error) {
	val, err := cr.ReadString(key)
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(val)
}

func (cr *goGitConfigReader) ReadString(key string) (string, error) {
	cfg, err := cr.getConfig()
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

func (cr *goGitConfigReader) ReadTimestamp(key string) (time.Time, error) {
	value, err := cr.ReadString(key)
	if err != nil {
		return time.Time{}, err
	}
	return ParseTimestamp(value)
}

var _ ConfigWrite = &goGitConfigWriter{}

// Only works for the local config as go-git only support that
type goGitConfigWriter struct {
	repo *gogit.Repository
}

func (cw *goGitConfigWriter) StoreString(key, value string) error {
	cfg, err := cw.repo.Config()
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
		subsection := strings.Join(split[1:len(split)-1], ".")
		option := split[len(split)-1]
		cfg.Raw.Section(section).Subsection(subsection).SetOption(option, value)
	}

	return cw.repo.SetConfig(cfg)
}

func (cw *goGitConfigWriter) StoreTimestamp(key string, value time.Time) error {
	return cw.StoreString(key, strconv.Itoa(int(value.Unix())))
}

func (cw *goGitConfigWriter) StoreBool(key string, value bool) error {
	return cw.StoreString(key, strconv.FormatBool(value))
}

func (cw *goGitConfigWriter) RemoveAll(keyPrefix string) error {
	cfg, err := cw.repo.Config()
	if err != nil {
		return err
	}

	split := strings.Split(keyPrefix, ".")

	switch {
	case keyPrefix == "":
		cfg.Raw.Sections = nil
		// warning: this does not actually remove everything as go-git config hold
		// some entries in multiple places (cfg.User ...)
	case len(split) == 1:
		if cfg.Raw.HasSection(split[0]) {
			cfg.Raw.RemoveSection(split[0])
		} else {
			return fmt.Errorf("invalid key prefix")
		}
	default:
		if !cfg.Raw.HasSection(split[0]) {
			return fmt.Errorf("invalid key prefix")
		}
		section := cfg.Raw.Section(split[0])
		rest := strings.Join(split[1:], ".")

		ok := false
		if section.HasSubsection(rest) {
			section.RemoveSubsection(rest)
			ok = true
		}
		if section.HasOption(rest) {
			section.RemoveOption(rest)
			ok = true
		}
		if !ok {
			return fmt.Errorf("invalid key prefix")
		}
	}

	return cw.repo.SetConfig(cfg)
}
