package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

const (
	defaultName = "default"
)

var (
	bridgeConfigureName   string
	bridgeConfigureTarget string
	bridgeParams          core.BridgeParams
)

func runBridgeConfigure(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	if (bridgeParams.TokenStdin || bridgeParams.Token != "" || bridgeParams.TokenId != "") &&
		(bridgeConfigureName == "" || bridgeConfigureTarget == "") {
		return fmt.Errorf("you must provide a bridge name and target to configure a bridge with a token")
	}

	if bridgeConfigureTarget == "" {
		bridgeConfigureTarget, err = promptTarget()
		if err != nil {
			return err
		}
	}

	if bridgeConfigureName == "" {
		bridgeConfigureName, err = promptName(repo)
		if err != nil {
			return err
		}
	}

	b, err := bridge.NewBridge(backend, bridgeConfigureTarget, bridgeConfigureName)
	if err != nil {
		return err
	}

	err = b.Configure(bridgeParams)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully configured bridge: %s\n", bridgeConfigureName)
	return nil
}

func promptTarget() (string, error) {
	targets := bridge.Targets()

	for {
		for i, target := range targets {
			fmt.Printf("[%d]: %s\n", i+1, target)
		}
		fmt.Printf("target: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')

		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)

		index, err := strconv.Atoi(line)
		if err != nil || index <= 0 || index > len(targets) {
			fmt.Println("invalid input")
			continue
		}

		return targets[index-1], nil
	}
}

func promptName(repo repository.RepoCommon) (string, error) {
	defaultExist := core.BridgeExist(repo, defaultName)

	for {
		if defaultExist {
			fmt.Printf("name: ")
		} else {
			fmt.Printf("name [%s]: ", defaultName)
		}

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)

		name := line
		if defaultExist && name == "" {
			continue
		}

		if name == "" {
			name = defaultName
		}

		if !core.BridgeExist(repo, name) {
			return name, nil
		}

		fmt.Println("a bridge with the same name already exist")
	}
}

var bridgeConfigureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure a new bridge.",
	Long: `	Configure a new bridge by passing flags or/and using interactive terminal prompts. You can avoid all the terminal prompts by passing all the necessary flags to configure your bridge.
	Repository configuration can be made by passing either the --url flag or the --project and --owner flags. If the three flags are provided git-bug will use --project and --owner flags.
	Token configuration can be directly passed with the --token flag or in the terminal prompt. If you don't already have one you can use the interactive procedure to generate one.`,
	Example: `# Interactive example
[1]: github
[2]: launchpad-preview
target: 1
name [default]: default

Detected projects:
[1]: github.com/a-hilaly/git-bug
[2]: github.com/MichaelMure/git-bug

[0]: Another project

Select option: 1

[1]: user provided token
[2]: interactive token creation
Select option: 1

You can generate a new token by visiting https://github.com/settings/tokens.
Choose 'Generate new token' and set the necessary access scope for your repository.

The access scope depend on the type of repository.
Public:
	- 'public_repo': to be able to read public repositories
Private:
	- 'repo'       : to be able to read private repositories

Enter token: 87cf5c03b64029f18ea5f9ca5679daa08ccbd700
Successfully configured bridge: default

# For GitHub
git bug bridge configure \
    --name=default \
    --target=github \
    --owner=$(OWNER) \
    --project=$(PROJECT) \
    --token=$(TOKEN)

# For Launchpad
git bug bridge configure \
    --name=default \
    --target=launchpad-preview \
	--url=https://bugs.launchpad.net/ubuntu/

# For Gitlab
git bug bridge configure \
    --name=default \
    --target=github \
    --url=https://github.com/michaelmure/git-bug \
    --token=$(TOKEN)`,
	PreRunE: loadRepo,
	RunE:    runBridgeConfigure,
}

func init() {
	bridgeCmd.AddCommand(bridgeConfigureCmd)
	bridgeConfigureCmd.Flags().StringVarP(&bridgeConfigureName, "name", "n", "", "A distinctive name to identify the bridge")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeConfigureTarget, "target", "t", "",
		fmt.Sprintf("The target of the bridge. Valid values are [%s]", strings.Join(bridge.Targets(), ",")))
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.URL, "url", "u", "", "The URL of the target repository")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.Owner, "owner", "o", "", "The owner of the target repository")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.Token, "token", "T", "", "The authentication token for the API")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.TokenId, "token-id", "i", "", "The authentication token identifier for the API")
	bridgeConfigureCmd.Flags().BoolVar(&bridgeParams.TokenStdin, "token-stdin", false, "Will read the token from stdin and ignore --token")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.Project, "project", "p", "", "The name of the target repository")
	bridgeConfigureCmd.Flags().SortFlags = false
}
