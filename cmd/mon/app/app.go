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

	"github.com/olive-io/olive/mon"
	"github.com/olive-io/olive/mon/config"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/pkg/version"
)

func NewMonCommand(stdout, stderr io.Writer) *cobra.Command {
	cfg := config.NewConfig()
	app := &cobra.Command{
		Use:     "olive-mon",
		Short:   "a component of olive",
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
		if err := PrintUsage(stderr); err != nil {
			return err
		}

		return PrintFlags(stderr)
	})

	app.ResetFlags()
	flags := app.PersistentFlags()
	flags.AddFlagSet(cfg.FlagSet())

	return app
}

func setup(cfg config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	ms, err := mon.New(cfg)
	if err != nil {
		return err
	}

	ctx := genericserver.SetupSignalContext()
	return ms.Start(ctx)
}

func PrintUsage(out io.Writer) error {
	_, err := fmt.Fprintln(out, usageLine)
	return err
}

func PrintFlags(out io.Writer) error {
	_, err := fmt.Fprintln(out, flagsLine)
	return err
}
