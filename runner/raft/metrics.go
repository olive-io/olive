// MIT License
//
// Copyright (c) 2023 Lack
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
		Name:      "bpmn_definitions",
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
