package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/99designs/gqlgen/codegen"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var genCmd = cli.Command{
	Name:  "generate",
	Usage: "generate a graphql server based on schema",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "verbose, v", Usage: "show logs"},
		cli.StringFlag{Name: "config, c", Usage: "the config filename"},
	},
	Action: func(ctx *cli.Context) {
		var config *codegen.Config
		var err error
		if configFilename := ctx.String("config"); configFilename != "" {
			config, err = codegen.LoadConfig(configFilename)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		} else {
			config, err = codegen.LoadConfigFromDefaultLocations()
			if os.IsNotExist(errors.Cause(err)) {
				config = codegen.DefaultConfig()
			} else if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		}

		for _, filename := range config.SchemaFilename {
			var schemaRaw []byte
			schemaRaw, err = ioutil.ReadFile(filename)
			if err != nil {
				fmt.Fprintln(os.Stderr, "unable to open schema: "+err.Error())
				os.Exit(1)
			}
			config.SchemaStr[filename] = string(schemaRaw)
		}

		if err = config.Check(); err != nil {
			fmt.Fprintln(os.Stderr, "invalid config format: "+err.Error())
			os.Exit(1)
		}

		err = codegen.Generate(*config)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(2)
		}
	},
}
