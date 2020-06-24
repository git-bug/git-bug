package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	identity2 "github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
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

	ids := backend.AllIdentityIds()
	var users []*cache.IdentityExcerpt
	for _, id := range ids {
		user, err := backend.ResolveIdentityExcerpt(id)
		if err != nil {
			return err
		}
		users = append(users, user)
	}

	switch userLsOutputFormat {
	case "org-mode":
		return userLsOrgmodeFormatter(users)
	case "json":
		return userLsJsonFormatter(users)
	case "plain":
		return userLsPlainFormatter(users)
	case "default":
		return userLsDefaultFormatter(users)
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

func NewJSONIdentity(id interface{}) (JSONIdentity, error) {
	switch id.(type) {
	case *cache.IdentityExcerpt:
		i := id.(*cache.IdentityExcerpt)
		return JSONIdentity{
			i.Id.String(),
			i.Id.Human(),
			i.Name,
			i.Login,
		}, nil
	case identity2.Interface:
		i := id.(identity2.Interface)
		return JSONIdentity{
			i.Id().String(),
			i.Id().Human(),
			i.Name(),
			i.Login(),
		}, nil
	case cache.LegacyAuthorExcerpt:
		i := id.(cache.LegacyAuthorExcerpt)
		return JSONIdentity{
			nil,
			nil,
			i.Name,
			i.Login,
		}, nil
	default:
		return JSONIdentity{}, errors.New(fmt.Sprintf("Inconvertible type, attempting to convert type %s to type %s.", reflect.TypeOf(id).String(), reflect.TypeOf(JSONIdentity{}).String()))
	}
}

func userLsPlainFormatter(users []*cache.IdentityExcerpt) error {
	for _, user := range users {
		fmt.Printf("%s %s\n",
			user.Id.Human(),
			user.DisplayName(),
		)
	}

	return nil
}

func userLsDefaultFormatter(users []*cache.IdentityExcerpt) error {
	for _, user := range users {
		fmt.Printf("%s %s\n",
			colors.Cyan(user.Id.Human()),
			user.DisplayName(),
		)
	}

	return nil
}

func userLsJsonFormatter(users []*cache.IdentityExcerpt) error {
	jsonUsers := []JSONIdentity{}
	for _, user := range users {
		jsonUser, err := NewJSONIdentity(user)
		if err != nil {
			return err
		}
		jsonUsers = append(jsonUsers, jsonUser)
	}

	jsonObject, _ := json.MarshalIndent(jsonUsers, "", "    ")
	fmt.Printf("%s\n", jsonObject)
	return nil
}

func userLsOrgmodeFormatter(users []*cache.IdentityExcerpt) error {
	for _, user := range users {
		fmt.Printf("* %s %s\n",
			user.Id.Human(),
			user.DisplayName(),
		)
	}

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
		"Select the output formatting style. Valid values are [default,plain,json,org-mode]")
}
