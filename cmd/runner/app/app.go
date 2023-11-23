// Copyright 2023 Lack (xingyys@gmail.com).
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
	"os"
	"os/signal"

	"github.com/olive-io/olive/pkg/signalutil"
	"github.com/olive-io/olive/pkg/version"
	"github.com/olive-io/olive/runner"
	"github.com/spf13/cobra"
)

func NewRunnerCommand() *cobra.Command {
	app := &cobra.Command{
		Use:           "olive-runner",
		Short:         "bpmn runner of olive cloud",
		Version:       version.Version,
		RunE:          setupRunner,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	app.ResetFlags()
	flags := app.PersistentFlags()
	runner.AddFlagSet(flags)

	return app
}

func setupRunner(cmd *cobra.Command, args []string) error {
	flags := cmd.PersistentFlags()
	cfg, err := runner.NewConfigFromFlagSet(flags)
	if err != nil {
		return err
	}

	if err = cfg.Validate(); err != nil {
		return err
	}

	oliveRunner, err := runner.NewRunner(cfg)
	if err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signalutil.Shutdown()...)

	if err = oliveRunner.Start(); err != nil {
		return err
	}

	select {
	// wait on kill signal
	case <-ch:
	}

	oliveRunner.Stop()

	return nil
}
