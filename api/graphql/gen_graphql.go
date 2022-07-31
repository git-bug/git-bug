//go:build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
)

func main() {
	fmt.Println("Generating graphql code ...")

	log.SetOutput(ioutil.Discard)

	cfg, err := config.LoadConfigFromDefaultLocations()
	if os.IsNotExist(errors.Cause(err)) {
		cfg = config.DefaultConfig()
	} else if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	if err = api.Generate(cfg); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}
