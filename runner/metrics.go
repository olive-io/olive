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

package runner

import (
	goruntime "runtime"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/olive-io/olive/pkg/version"
)

var (
	currentVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "runner",
		Name:      "version",
		Help:      "Which version is running. 1 for 'runner_version' label with current version.",
	},
		[]string{"runner_version"})
	currentGoVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "runner",
		Name:      "go_version",
		Help:      "Which Go version runner is running with. 1 for 'runner_go_version' label with current version.",
	},
		[]string{"runner_go_version"})
)

func init() {
	prometheus.MustRegister(currentVersion)
	prometheus.MustRegister(currentGoVersion)

	currentVersion.With(prometheus.Labels{
		"runner_version": version.Version,
	}).Set(1)
	currentGoVersion.With(prometheus.Labels{
		"runner_go_version": goruntime.Version(),
	}).Set(1)
}
