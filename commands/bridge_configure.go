package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MichaelMure/git-bug/bridge/core"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

const (
	defaultName = "default"
)

var (
	name         string
	target       string
	bridgeParams core.BridgeParams
)

func runBridgeConfigure(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	if target == "" {
		target, err = promptTarget()
		if err != nil {
			return err
		}
	}

	if name == "" {
		name, err = promptName()
		if err != nil {
			return err
		}
	}

	b, err := bridge.NewBridge(backend, target, name)
	if err != nil {
		return err
	}

	err = b.Configure(bridgeParams)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully configured bridge: %s\n", name)
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

		line = strings.TrimRight(line, "\n")

		index, err := strconv.Atoi(line)
		if err != nil || index <= 0 || index > len(targets) {
			fmt.Println("invalid input")
			continue
		}

		return targets[index-1], nil
	}
}

func promptName() (string, error) {
	fmt.Printf("name [%s]: ", defaultName)

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimRight(line, "\n")

	if line == "" {
		return defaultName, nil
	}

	return line, nil
}

var bridgeConfigureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure a new bridge.",
	Long: `	Configure a new bridge by passing flags or/and using interactive terminal prompts. You can avoid all the terminal prompts by passing all the necessary flags to configure your bridge.
	Repository configuration can be made by passing or the --url flag or the --project and/or --owner flags. If the three flags are provided git-bug will use --project and --owner flags.
	Token configuration can be made by passing it in the --token flag or in the terminal prompt. If you don't already have one you can use terminal prompt to login and generate it directly.
	For Github and Gitlab bridges, git-bug need a token to export and import issues, comments and editions for public and private repositories.
	For Launchpad bridges, git-bug for now supports only public repositories and you only need --project or --url flag to configure it.`,
	Example: `# For Github
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
    --url=https://bugs.launchpad.net/ubuntu/`,
	PreRunE: loadRepo,
	RunE:    runBridgeConfigure,
}

func init() {
	bridgeCmd.AddCommand(bridgeConfigureCmd)
	bridgeConfigureCmd.Flags().StringVarP(&name, "name", "n", "", "A distinctive name to identify the bridge")
	bridgeConfigureCmd.Flags().StringVarP(&target, "target", "t", "",
		fmt.Sprintf("The target of the bridge. Valid values are [%s]", strings.Join(bridge.Targets(), ",")))
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.URL, "url", "u", "", "The URL of the target repository")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.Owner, "owner", "o", "", "The owner of the target repository")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.Token, "token", "T", "", "The authentication token for the API")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.Project, "project", "p", "", "The name of the target repository")
	bridgeConfigureCmd.Flags().SortFlags = false
}
