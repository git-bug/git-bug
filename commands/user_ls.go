package commands

import (
	"encoding/json"
	"fmt"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

var (
	userLsOutputFormat string
)

func runUserLs(_ *cobra.Command, _ []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	switch userLsOutputFormat {
	case "json":
		return userLsJsonFormatter(backend)
	case "plain":
		return userLsPlainFormatter(backend)
	case "default":
		return userLsDefaultFormatter(backend)
	default:
		return fmt.Errorf("unknown format %s", userLsOutputFormat)
	}
}

type JSONIdentity struct {
	Id      string `json:"id"`
	HumanId string `json:"human_id"`
	Name    string `json:"name"`
	Login   string `json:"login"`
}

func userLsPlainFormatter(backend *cache.RepoCache) error {
	for _, id := range backend.AllIdentityIds() {
		i, err := backend.ResolveIdentityExcerpt(id)
		if err != nil {
			return err
		}

		fmt.Printf("%s %s\n",
			i.Id.Human(),
			i.DisplayName(),
		)
	}

	return nil
}

func userLsDefaultFormatter(backend *cache.RepoCache) error {
	for _, id := range backend.AllIdentityIds() {
		i, err := backend.ResolveIdentityExcerpt(id)
		if err != nil {
			return err
		}

		fmt.Printf("%s %s\n",
			colors.Cyan(i.Id.Human()),
			i.DisplayName(),
		)
	}

	return nil
}

func userLsJsonFormatter(backend *cache.RepoCache) error {
	users := []JSONIdentity{}
	for _, id := range backend.AllIdentityIds() {
		i, err := backend.ResolveIdentityExcerpt(id)
		if err != nil {
			return err
		}

		users = append(users, JSONIdentity{
			i.Id.String(),
			i.Id.Human(),
			i.Name,
			i.Login,
		})
	}

	jsonObject, _ := json.MarshalIndent(users, "", "    ")
	fmt.Printf("%s\n", jsonObject)
	return nil
}

var userLsCmd = &cobra.Command{
	Use:     "ls",
	Short:   "List identities.",
	PreRunE: loadRepo,
	RunE:    runUserLs,
}

func init() {
	userCmd.AddCommand(userLsCmd)
	userLsCmd.Flags().SortFlags = false
	userLsCmd.Flags().StringVarP(&userLsOutputFormat, "format", "f", "default",
		"Select the output formatting style. Valid values are [default,plain,json]")
}
