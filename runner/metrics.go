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

package runner

import (
	goruntime "runtime"

	"github.com/olive-io/olive/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
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
		"runner_version": version.GoV(),
	}).Set(1)
	currentGoVersion.With(prometheus.Labels{
		"runner_go_version": goruntime.Version(),
	}).Set(1)
}
