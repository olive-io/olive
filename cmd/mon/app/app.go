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

	"github.com/olive-io/olive/apis/version"
	"github.com/olive-io/olive/mon/options"
)

// NewMonServer provides a CLI handler for 'start master' command
// with a default ServerOptions.
func NewMonServer(defaults *options.ServerOptions, stopCh <-chan struct{}) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Use:     "olive-mon",
		Short:   "a component of olive",
		Version: version.Version,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.StartMonServer(stopCh); err != nil {
				return err
			}
			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.SetOut(o.StdOut)
	cmd.SetErr(o.StdErr)

	//cmd.SetUsageFunc(func(cmd *cobra.Command) error {
	//	if err := PrintUsage(o.StdOut); err != nil {
	//		return err
	//	}
	//
	//	return PrintFlags(o.StdOut)
	//})

	flags := cmd.Flags()
	o.EmbedEtcdOptions.AddFlags(flags)
	o.RecommendedOptions.AddFlags(flags)

	return cmd
}

func PrintUsage(out io.Writer) error {
	_, err := fmt.Fprintln(out, usageline)
	return err
}

func PrintFlags(out io.Writer) error {
	_, err := fmt.Fprintln(out, flagsline)
	return err
}
