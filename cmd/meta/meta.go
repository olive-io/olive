package main

import (
	"os"

	"github.com/olive-io/olive/cmd/meta/app"
	"github.com/olive-io/olive/pkg/component-base/cli"
)

func main() {
	command := app.NewMetaCommand()
	code := cli.Run(command)
	os.Exit(code)
}
