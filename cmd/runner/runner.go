package main

import (
	"os"

	"github.com/olive-io/olive/cmd/runner/app"
	"github.com/olive-io/olive/pkg/component-base/cli"
)

func main() {
	command := app.NewRunnerCommand()
	code := cli.Run(command)
	os.Exit(code)
}
