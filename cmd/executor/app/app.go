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
	"github.com/olive-io/olive/execute/server"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/pkg/version"
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
	server.AddFlagSet(flags)

	return app
}

func setupExecutor(cmd *cobra.Command, args []string) error {
	stopc := genericserver.SetupSignalHandler()
	flags := cmd.PersistentFlags()
	cfg, err := server.NewConfigFromFlagSet(flags)
	if err != nil {
		return err
	}

	if err = cfg.Validate(); err != nil {
		return err
	}

	executor, err := server.NewExecutor(cfg)
	if err != nil {
		return err
	}

	return executor.Start(stopc)
}
