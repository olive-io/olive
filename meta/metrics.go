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

package meta

import (
	goruntime "runtime"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/olive-io/olive/pkg/version"
)

var (
	currentVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "meta",
		Name:      "version",
		Help:      "Which version is running. 1 for 'meta_version' label with current version.",
	},
		[]string{"meta_version"})
	currentGoVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "meta",
		Name:      "go_version",
		Help:      "Which Go version meta is running with. 1 for 'meta_go_version' label with current version.",
	},
		[]string{"meta_go_version"})
)

func init() {
	prometheus.MustRegister(currentVersion)
	prometheus.MustRegister(currentGoVersion)

	currentVersion.With(prometheus.Labels{
		"meta_version": version.Version,
	}).Set(1)
	currentGoVersion.With(prometheus.Labels{
		"meta_go_version": goruntime.Version(),
	}).Set(1)
}
