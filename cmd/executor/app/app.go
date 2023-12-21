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
	"os"
	"os/signal"

	"github.com/olive-io/olive/executor"
	"github.com/olive-io/olive/pkg/signalutil"
	"github.com/olive-io/olive/pkg/version"
	"github.com/olive-io/olive/runner"
	"github.com/spf13/cobra"
)

func NewExecutorCommand() *cobra.Command {
	app := &cobra.Command{
		Use:           "olive-executor",
		Short:         "bpmn executor of olive cloud",
		Version:       version.Version,
		RunE:          setupExecutor,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	app.ResetFlags()
	flags := app.PersistentFlags()
	runner.AddFlagSet(flags)

	return app
}

func setupExecutor(cmd *cobra.Command, args []string) error {
	flags := cmd.PersistentFlags()
	cfg, err := executor.NewConfigFromFlagSet(flags)
	if err != nil {
		return err
	}

	if err = cfg.Validate(); err != nil {
		return err
	}

	oliveExecutor, err := executor.NewExecutor(cfg)
	if err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signalutil.Shutdown()...)

	if err = oliveExecutor.Start(); err != nil {
		return err
	}

	select {
	// wait on kill signal
	case <-ch:
	}

	oliveExecutor.Stop()

	return nil
}
