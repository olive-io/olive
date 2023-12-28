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

package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	cliflag "github.com/olive-io/olive/pkg/component-base/cli/flags"
	"github.com/olive-io/olive/pkg/component-base/logs"
)

// Run provides the common boilerplate code around executing a cobra command.
// For example, it ensures that logging is set up properly. Logging
// flags get added to the command line if not added already. Flags get normalized
// so that help texts show them with hyphens. Underscores are accepted
// as alternative for the command parameters.
//
// Run tries to be smart about how to print errors that are returned by the
// command: before logging is known to be set up, it prints them as plain text
// to stderr. This covers command line flag parse errors and unknown commands.
// Afterwards it logs them. This covers runtime errors.
func Run(cmd *cobra.Command) int {
	if logsInitialized, err := run(cmd); err != nil {
		if !logsInitialized {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			slog.Error("command failed", slog.String("err", err.Error()))
		}
		return 1
	}
	return 0
}

// RunNoErrOutput is a version of Run which returns the cobra command error
// instead of printing it.
func RunNoErrOutput(cmd *cobra.Command) error {
	_, err := run(cmd)
	return err
}

func run(cmd *cobra.Command) (logsInitialized bool, err error) {
	cmd.SetGlobalNormalizationFunc(cliflag.WordSepNormalizeFunc)

	// When error printing is enabled for the Cobra command, a flag parse
	// error gets printed first, then optionally the often long usage
	// text. This is very unreadable in a console because the last few
	// lines that will be visible on screen don't include the error.
	//
	// The recommendation from #sig-cli was to print the usage text, then
	// the error. We implement this consistently for all commands here.
	// However, we don't want to print the usage text when command
	// execution fails for other reasons than parsing. We detect this via
	// the FlagParseError callback.
	//
	// Some commands, like kubectl, already deal with this themselves.
	// We don't change the behavior for those.
	if !cmd.SilenceUsage {
		cmd.SilenceUsage = true
		cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
			// Re-enable usage printing.
			c.SilenceUsage = false
			return err
		})
	}

	// In all cases error printing is done below.
	cmd.SilenceErrors = true

	// This is idempotent.
	//logs.AddFlags(cmd.PersistentFlags())

	// Inject logs.InitLogs after command line parsing into one of the
	// PersistentPre* functions.
	switch {
	case cmd.PersistentPreRun != nil:
		pre := cmd.PersistentPreRun
		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			logs.InitLogs()
			logsInitialized = true
			pre(cmd, args)
		}
	case cmd.PersistentPreRunE != nil:
		pre := cmd.PersistentPreRunE
		cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			logs.InitLogs()
			logsInitialized = true
			return pre(cmd, args)
		}
	default:
		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			logs.InitLogs()
			logsInitialized = true
		}
	}

	err = cmd.Execute()
	return
}
