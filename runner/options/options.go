/*
Copyright 2024 The olive Authors

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

package options

import (
	"io"

	"github.com/spf13/pflag"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/olive-io/olive/apis"
	"github.com/olive-io/olive/pkg/cliutil/flags"
	"github.com/olive-io/olive/pkg/logutil"
	"github.com/olive-io/olive/runner"
)

type DefaultOptions struct {
	*runner.Config

	Stdout io.Writer
	Stderr io.Writer
}

func NewRunnerOptions(stdout, stderr io.Writer) *DefaultOptions {
	return &DefaultOptions{
		Config: runner.NewConfig(),
		Stdout: stdout,
		Stderr: stderr,
	}
}

func (o *DefaultOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Name, "name", o.Name, "Set the name of olive-runner.")
	fs.StringVar(&o.DataDir, "data-dir", o.DataDir, "Path to the data directory.")
	fs.StringVar(&o.ConfigPath, "oliveconfig", o.ConfigPath, "Set the file path from configuration the cluster of olive-mon")
	fs.StringVar(&o.ListenURL, "listen-url", o.ListenURL, "Set the URL to listen on for client traffic.")
	fs.StringVar(&o.AdvertiseURL, "advertise-url", o.AdvertiseURL, "Set advertise URL to listen on for client traffic.")

	// Backend
	fs.DurationVar(&o.BackendGCInterval, "backend-gc-interval", o.BackendGCInterval, "the maximum interval for the backend garbage collection.")

	// logging
	fs.Var(flags.NewUniqueStringsValue(logutil.DefaultLogOutput), "log-outputs",
		"Specify 'stdout' or 'stderr' to skip journald logging even when running under systemd, or list of comma separated output targets.")
	fs.StringVar(&o.LogLevel, "log-level", logutil.DefaultLogLevel,
		"Configures log level. Only supports debug, info, warn, error, panic, or fatal. Default 'info'.")
	fs.BoolVar(&o.EnableLogRotation, "enable-log-rotation", false,
		"Enable log rotation of a single log-outputs file target.")
	fs.StringVar(&o.LogRotationConfigJSON, "log-rotation-config-json", logutil.DefaultLogRotationConfig,
		"Configures log rotation if enabled with a JSON logger config. Default: MaxSize=100(MB), MaxAge=0(days,no limit), MaxBackups=0(no limit), LocalTime=false(UTC), Compress=false(gzip)")
}

func (o *DefaultOptions) Complete() error {
	var err error
	if err = o.Config.Complete(); err != nil {
		return err
	}
	return nil
}

func (o *DefaultOptions) Validate(args []string) error {
	errors := make([]error, 0)
	if err := o.Config.Validate(); err != nil {
		errors = append(errors, err)
	}
	return utilerrors.NewAggregate(errors)
}

func (o *DefaultOptions) StartRunner(stopc <-chan struct{}) error {
	config := o.Config
	r, err := runner.NewRunner(config, apis.Scheme)
	if err != nil {
		return err
	}

	return r.Start(stopc)
}
