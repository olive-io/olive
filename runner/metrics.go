/*
Copyright 2025 The olive Authors

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

package runner

import (
	goruntime "runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"

	"github.com/olive-io/olive/pkg/metrics"
)

var (
	DefaultNamespace = "olive"
	DefaultSubsystem = "runner"
)

var (
	currentVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
		Name:      "version",
		Help:      "Which version is running. 1 for 'runner_version' label with current version.",
	},
		[]string{"runner_version"})
	currentGoVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
		Name:      "go_version",
		Help:      "Which Go version runner is running with. 1 for 'runner_go_version' label with current version.",
	},
		[]string{"runner_go_version"})

	stepCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
		Name:      "step_count",
		Help:      "the number of Step",
	})

	stepCommitCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
		Name:      "step_commit_count",
		Help:      "the number of Step on commit",
	})

	stepRollbackCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
		Name:      "step_commit_count",
		Help:      "the number of Step on rollback",
	})

	stepDestroyCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
		Name:      "step_commit_count",
		Help:      "the number of Step on destroy",
	})
)

func init() {
	prometheus.MustRegister(currentVersion)
	prometheus.MustRegister(currentGoVersion)
	prometheus.MustRegister(stepCounter)
	prometheus.MustRegister(stepCommitCounter)
	prometheus.MustRegister(stepRollbackCounter)
	prometheus.MustRegister(stepDestroyCounter)

	currentVersion.With(prometheus.Labels{
		"runner_version": version.Version,
	}).Set(1)

	currentGoVersion.With(prometheus.Labels{
		"runner_go_version": goruntime.Version(),
	}).Set(1)
}
