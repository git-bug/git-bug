package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core"
)

var (
	bridgeTokenTarget string
)

func runBridgeTokenAdd(cmd *cobra.Command, args []string) error {
	var value string

	if bridgeTokenTarget == "" {
		return fmt.Errorf("token target is required")
	}

	if !core.TargetExist(bridgeTokenTarget) {
		return fmt.Errorf("unknown target")
	}

	if len(args) == 1 {
		value = args[0]
	} else {
		// Read from Stdin
		if isatty.IsTerminal(os.Stdin.Fd()) {
			fmt.Println("Enter the token:")
		}
		reader := bufio.NewReader(os.Stdin)
		raw, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading from stdin: %v", err)
		}
		value = strings.TrimSuffix(raw, "\n")
	}

	token := core.NewToken(value, bridgeTokenTarget)
	if err := token.Validate(); err != nil {
		return errors.Wrap(err, "invalid token")
	}

	err := core.StoreToken(repo, token)
	if err != nil {
		return err
	}

	fmt.Printf("token %s added\n", token.ID())
	return nil
}

var bridgeTokenAddCmd = &cobra.Command{
	Use:     "add",
	Short:   "Store a new token",
	PreRunE: loadRepo,
	RunE:    runBridgeTokenAdd,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	bridgeTokenCmd.AddCommand(bridgeTokenAddCmd)
	bridgeTokenAddCmd.Flags().StringVarP(&bridgeTokenTarget, "target", "t", "",
		fmt.Sprintf("The target of the bridge. Valid values are [%s]", strings.Join(bridge.Targets(), ",")))
	bridgeTokenAddCmd.Flags().SortFlags = false
}
