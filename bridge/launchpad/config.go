package launchpad

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
)

var ErrBadProjectURL = errors.New("bad Launchpad project URL")

func (Launchpad) ValidParams() map[string]interface{} {
	return map[string]interface{}{
		"URL":     nil,
		"Project": nil,
	}
}

func (l *Launchpad) Configure(repo *cache.RepoCache, params core.BridgeParams, interactive bool) (core.Configuration, error) {
	var err error
	var project string

	switch {
	case params.Project != "":
		project = params.Project
	case params.URL != "":
		// get project name from url
		project, err = splitURL(params.URL)
	default:
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the project name with the --project option.")
		}
		// get project name from terminal prompt
		project, err = input.Prompt("Launchpad project name", "project name", input.Required)
	}

	if err != nil {
		return nil, err
	}

	// verify project
	ok, err := validateProject(project)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("project doesn't exist")
	}

	conf := make(core.Configuration)
	conf[core.ConfigKeyTarget] = target
	conf[confKeyProject] = project

	err = l.ValidateConfig(conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (*Launchpad) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.ConfigKeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.ConfigKeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}
	if _, ok := conf[confKeyProject]; !ok {
		return fmt.Errorf("missing %s key", confKeyProject)
	}

	return nil
}

func validateProject(project string) (bool, error) {
	url := fmt.Sprintf("%s/%s", apiRoot, project)

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}

	_ = resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// extract project name from url
func splitURL(url string) (string, error) {
	re := regexp.MustCompile(`launchpad\.net[\/:]([^\/]*[a-z]+)`)

	res := re.FindStringSubmatch(url)
	if res == nil {
		return "", ErrBadProjectURL
	}

	return res[1], nil
}
