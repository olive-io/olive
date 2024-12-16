/*
Copyright 2023 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package app

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/olive-io/olive/api"
	corev1 "github.com/olive-io/olive/api/types/core/v1"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/pkg/version"
	"github.com/olive-io/olive/runner"
	"github.com/olive-io/olive/runner/config"
)

func NewRunnerCommand(stdout, stderr io.Writer) *cobra.Command {
	cfg := config.NewConfig()

	app := &cobra.Command{
		Use:     "olive-runner",
		Short:   "bpmn runner of olive cloud",
		Version: version.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cfg.Parse(); err != nil {
				return err
			}

			return setup(*cfg)
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	app.SetUsageFunc(func(cmd *cobra.Command) error {
		if err := PrintUsage(stdout); err != nil {
			return err
		}

		return PrintFlags(stdout)
	})
	app.SetErr(stderr)

	app.ResetFlags()
	flags := app.PersistentFlags()
	flags.AddFlagSet(cfg.FlagSet())

	return app
}

func setup(cfg config.Config) error {
	var err error
	stopc := genericserver.SetupSignalHandler()
	if err = cfg.Validate(); err != nil {
		return err
	}

	scheme := api.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return err
	}

	server, err := runner.NewRunner(&cfg, scheme)
	if err != nil {
		return err
	}

	return server.Start(stopc)
}

func PrintUsage(out io.Writer) error {
	_, err := fmt.Fprintln(out, usageline)
	return err
}

func PrintFlags(out io.Writer) error {
	_, err := fmt.Fprintln(out, flagsline)
	return err
}
