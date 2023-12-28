// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/olive-io/olive/meta"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/pkg/version"
)

func NewMetaCommand(stdout, stderr io.Writer) *cobra.Command {
	cfg := meta.NewConfig()

	app := &cobra.Command{
		Use:     "olive-meta",
		Short:   "a component of olive",
		Version: version.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cfg.Parse(); err != nil {
				return err
			}

			return setup(cfg)
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	app.SetUsageFunc(func(cmd *cobra.Command) error {
		if err := printUsage(stderr); err != nil {
			return err
		}

		return printFlags(stderr)
	})

	app.ResetFlags()
	flags := app.PersistentFlags()
	flags.AddFlagSet(cfg.FlagSet())

	return app
}

func setup(cfg meta.Config) error {
	stopc := genericserver.SetupSignalHandler()
	if err := cfg.Validate(); err != nil {
		return err
	}

	ms, err := meta.NewServer(cfg)
	if err != nil {
		return err
	}

	return ms.Start(stopc)
}

func printUsage(out io.Writer) error {
	_, err := fmt.Fprintln(out, usageline)
	return err
}

func printFlags(out io.Writer) error {
	_, err := fmt.Fprintln(out, flagsline)
	return err
}
