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

package metrics

import (
	goruntime "runtime"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/olive-io/olive/pkg/metrics"
	"github.com/olive-io/olive/pkg/version"
)

const (
	DefaultNamespace = "olive"
	DefaultSubsystem = "runner"
)

var fs = newSets()

type featureSets struct {
	sync.RWMutex
	features map[string][]string
	gauges   map[string]prometheus.Gauge
}

func newSets() *featureSets {
	sets := &featureSets{
		features: make(map[string][]string),
		gauges:   make(map[string]prometheus.Gauge),
	}
	return sets
}

func (fs *featureSets) addFeature(name, help string, values ...string) error {
	features := values
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   DefaultNamespace,
		Subsystem:   DefaultSubsystem,
		Name:        name,
		Help:        help,
		ConstLabels: map[string]string{"features": strings.Join(features, ",")},
	})
	gauge.Set(1)

	fs.Lock()
	fs.gauges[name] = gauge
	fs.features[name] = features
	fs.Unlock()

	return prometheus.Register(gauge)
}

func (fs *featureSets) getFeatures() map[string]string {
	fs.RLock()
	defer fs.RUnlock()
	outs := map[string]string{}
	for key, values := range fs.features {
		outs[key] = strings.Join(values, ",")
	}
	return outs
}

func AddFeature(name, help string, values ...string) error {
	return fs.addFeature(name, help, values...)
}

func GetFeatures() map[string]string {
	return fs.getFeatures()
}

var (
	currentVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
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

	ProcessCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
		Name:      "definitions",
		Help:      "the number of processes",
	})

	EventCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
		Name:      "definitions",
		Help:      "the number of bpmn events",
	})

	TaskCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: DefaultSubsystem,
		Name:      "definitions",
		Help:      "the number of bpmn tasks",
	})
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
