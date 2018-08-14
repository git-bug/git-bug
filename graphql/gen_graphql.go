// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/vektah/gqlgen/codegen"
)

func main() {
	current, err := os.Getwd()
	if err != nil {
		log.Fatal(err.Error())
	}

	os.Chdir(path.Join(current, "graphql"))

	fmt.Println("Generating graphql code ...")

	log.SetOutput(ioutil.Discard)

	config, err := codegen.LoadDefaultConfig()
	if err != nil {
		log.Fatal(err)
	}

	schemaRaw, err := ioutil.ReadFile(config.SchemaFilename)
	if err != nil {
		log.Fatal("unable to open schema: " + err.Error())
	}
	config.SchemaStr = string(schemaRaw)

	if err = config.Check(); err != nil {
		log.Fatal("invalid config format: " + err.Error())
	}

	err = codegen.Generate(*config)
	if err != nil {
		log.Fatal(err.Error())
	}
}
