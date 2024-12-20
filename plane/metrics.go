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

package plane

import (
	goruntime "runtime"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/olive-io/olive/apis/version"
)

var (
	currentVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "meta",
		Name:      "version",
		Help:      "Which version is running. 1 for 'meta_version' label with current version.",
	},
		[]string{"meta_version"})
	currentGITSHA = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "meta",
		Name:      "git_sha",
		Help:      "Which git sha is running. 1 for 'meta_git_sha' label with current git sha.",
	},
		[]string{"meta_git_sha"})
	currentGoVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "meta",
		Name:      "go_version",
		Help:      "Which Go version meta is running with. 1 for 'meta_go_version' label with current go version.",
	},
		[]string{"meta_go_version"})
)

func init() {
	prometheus.MustRegister(currentVersion)
	prometheus.MustRegister(currentGITSHA)
	prometheus.MustRegister(currentGoVersion)

	currentVersion.With(prometheus.Labels{
		"meta_version": version.Version,
	}).Set(1)
	currentGITSHA.With(prometheus.Labels{
		"meta_git_sha": version.GitSHA,
	})
	currentGoVersion.With(prometheus.Labels{
		"meta_go_version": goruntime.Version(),
	}).Set(1)
}
