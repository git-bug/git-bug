package launchpad

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
)

var ErrBadProjectURL = errors.New("bad Launchpad project URL")

const (
	target         = "launchpad-preview"
	keyProject     = "project"
	defaultTimeout = 60 * time.Second
)

func (l *Launchpad) Configure(repo *cache.RepoCache, params core.BridgeParams) (core.Configuration, error) {
	if params.TokenRaw != "" {
		fmt.Println("warning: token params are ineffective for a Launchpad bridge")
	}
	if params.Owner != "" {
		fmt.Println("warning: --owner is ineffective for a Launchpad bridge")
	}
	if params.BaseURL != "" {
		fmt.Println("warning: --base-url is ineffective for a Launchpad bridge")
	}

	conf := make(core.Configuration)
	var err error
	var project string

	switch {
	case params.Project != "":
		project = params.Project
	case params.URL != "":
		// get project name from url
		project, err = splitURL(params.URL)
	default:
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

	conf[core.ConfigKeyTarget] = target
	conf[keyProject] = project

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

	if _, ok := conf[keyProject]; !ok {
		return fmt.Errorf("missing %s key", keyProject)
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

	return resp.StatusCode == http.StatusOK, nil
}

// extract project name from url
func splitURL(url string) (string, error) {
	re, err := regexp.Compile(`launchpad\.net[\/:]([^\/]*[a-z]+)`)
	if err != nil {
		panic("regexp compile:" + err.Error())
	}

	res := re.FindStringSubmatch(url)
	if res == nil {
		return "", ErrBadProjectURL
	}

	return res[1], nil
}
