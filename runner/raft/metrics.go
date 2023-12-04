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

package raft

import (
	"fmt"

	"github.com/olive-io/olive/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RegionCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "runner",
		Name:      "region",
		Help:      "The counts of region on localhost",
	})

	LeaderCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "runner",
		Name:      "leader",
		Help:      "The counts of leader region on localhost",
	})

	DefinitionsCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "runner",
		Name:      "bpmn_definitions",
		Help:      "the total of bpmn definitions in all regions",
	})

	RunningDefinitionsCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "runner",
		Name:      "bpmn_running_definitions",
		Help:      "The counts of running bpmn definitions in all regions",
	})

	ProcessCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "runner",
		Name:      "bpmn_process",
		Help:      "The counts of running bpmn processes in all regions",
	})

	EventCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "runner",
		Name:      "bpmn_event",
		Help:      "The counts of running bpmn events in all regions",
	})

	TaskCounter = metrics.NewGauge(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "runner",
		Name:      "bpmn_task",
		Help:      "The counts of running bpmn tasks in all regions",
	})
)

func init() {
	prometheus.MustRegister(RegionCounter)
	prometheus.MustRegister(LeaderCounter)
	prometheus.MustRegister(DefinitionsCounter)
	prometheus.MustRegister(RunningDefinitionsCounter)
	prometheus.MustRegister(ProcessCounter)
	prometheus.MustRegister(EventCounter)
	prometheus.MustRegister(TaskCounter)
}

type regionMetrics struct {
	definition metrics.Gauge
	process    metrics.Gauge
	event      metrics.Gauge
	task       metrics.Gauge
}

func newRegionMetrics(id uint64) (*regionMetrics, error) {
	constLabels := prometheus.Labels{"region": fmt.Sprintf("%d", id)}
	definitionMetrics := metrics.NewGauge(prometheus.GaugeOpts{
		Namespace:   "olive",
		Subsystem:   "runner_region",
		Name:        "bpmn_definition",
		Help:        "The counts of running bpmn definitions the region",
		ConstLabels: constLabels,
	})
	processMetrics := metrics.NewGauge(prometheus.GaugeOpts{
		Namespace:   "olive",
		Subsystem:   "runner_region",
		Name:        "bpmn_process",
		Help:        "The counts of running bpmn processes the region",
		ConstLabels: constLabels,
	})
	eventMetrics := metrics.NewGauge(prometheus.GaugeOpts{
		Namespace:   "olive",
		Subsystem:   "runner_region",
		Name:        "bpmn_event",
		Help:        "The counts of running bpmn event the region",
		ConstLabels: constLabels,
	})
	taskMetrics := metrics.NewGauge(prometheus.GaugeOpts{
		Namespace:   "olive",
		Subsystem:   "runner_region",
		Name:        "bpmn_definition",
		Help:        "The counts of running bpmn tasks the region",
		ConstLabels: constLabels,
	})

	metric := &regionMetrics{
		definition: definitionMetrics,
		process:    processMetrics,
		event:      eventMetrics,
		task:       taskMetrics,
	}

	var err error
	if err = prometheus.Register(definitionMetrics); err != nil {
		return nil, err
	}
	if err = prometheus.Register(processMetrics); err != nil {
		return nil, err
	}
	if err = prometheus.Register(eventMetrics); err != nil {
		return nil, err
	}
	if err = prometheus.Register(taskMetrics); err != nil {
		return nil, err
	}

	return metric, nil
}
