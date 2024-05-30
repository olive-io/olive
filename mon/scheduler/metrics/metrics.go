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

package metrics

import (
	"sync"
	"time"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

const (
	// SchedulerSubsystem - subsystem name used by scheduler.
	SchedulerSubsystem = "scheduler"
)

// Below are possible values for the work and operation label.
const (
	// PrioritizingExtender - prioritizing extender work/operation label value.
	PrioritizingExtender = "prioritizing_extender"
	// Binding - binding work/operation label value.
	Binding = "binding"
)

// Below are possible values for the extension_point label.
const (
	PreFilter                          = "PreFilter"
	Filter                             = "Filter"
	PreFilterExtensionAddDefinition    = "PreFilterExtensionAddDefinition"
	PreFilterExtensionRemoveDefinition = "PreFilterExtensionRemoveDefinition"
	PostFilter                         = "PostFilter"
	PreScore                           = "PreScore"
	Score                              = "Score"
	ScoreExtensionNormalize            = "ScoreExtensionNormalize"
	PreBind                            = "PreBind"
	Bind                               = "Bind"
	PostBind                           = "PostBind"
	Reserve                            = "Reserve"
	Unreserve                          = "Unreserve"
	Permit                             = "Permit"
)

// All the histogram based metrics have 1ms as size for the smallest bucket.
var (
	scheduleAttempts = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "schedule_attempts_total",
			Help:           "Number of attempts to schedule definitions, by the result. 'unschedulable' means a definition could not be scheduled, while 'error' means an internal scheduler problem.",
			StabilityLevel: metrics.STABLE,
		}, []string{"result", "profile"})

	schedulingLatency = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "scheduling_attempt_duration_seconds",
			Help:           "Scheduling attempt latency in seconds (scheduling algorithm + binding)",
			Buckets:        metrics.ExponentialBuckets(0.001, 2, 15),
			StabilityLevel: metrics.STABLE,
		}, []string{"result", "profile"})
	SchedulingAlgorithmLatency = metrics.NewHistogram(
		&metrics.HistogramOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "scheduling_algorithm_duration_seconds",
			Help:           "Scheduling algorithm latency in seconds",
			Buckets:        metrics.ExponentialBuckets(0.001, 2, 15),
			StabilityLevel: metrics.ALPHA,
		},
	)
	PreemptionVictims = metrics.NewHistogram(
		&metrics.HistogramOpts{
			Subsystem: SchedulerSubsystem,
			Name:      "preemption_victims",
			Help:      "Number of selected preemption victims",
			// we think #victims>64 is pretty rare, therefore [64, +Inf) is considered a single bucket.
			Buckets:        metrics.ExponentialBuckets(1, 2, 7),
			StabilityLevel: metrics.STABLE,
		})
	PreemptionAttempts = metrics.NewCounter(
		&metrics.CounterOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "preemption_attempts_total",
			Help:           "Total preemption attempts in the cluster till now",
			StabilityLevel: metrics.STABLE,
		})
	pendingDefinitions = metrics.NewGaugeVec(
		&metrics.GaugeOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "pending_definitions",
			Help:           "Number of pending definitions, by the queue type. 'active' means number of definitions in activeQ; 'backoff' means number of definitions in backoffQ; 'unschedulable' means number of definitions in unschedulableDefinitions that the scheduler attempted to schedule and failed; 'gated' is the number of unschedulable definitions that the scheduler never attempted to schedule because they are gated.",
			StabilityLevel: metrics.STABLE,
		}, []string{"queue"})
	Goroutines = metrics.NewGaugeVec(
		&metrics.GaugeOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "goroutines",
			Help:           "Number of running goroutines split by the work they do such as binding.",
			StabilityLevel: metrics.ALPHA,
		}, []string{"operation"})

	// DefinitionSchedulingDuration is deprecated as of Kubernetes v1.28, and will be removed
	// in v1.31. Please use DefinitionSchedulingSLIDuration instead.
	DefinitionSchedulingDuration = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Subsystem: SchedulerSubsystem,
			Name:      "definition_scheduling_duration_seconds",
			Help:      "E2e latency for a definition being scheduled which may include multiple scheduling attempts.",
			// Start with 10ms with the last bucket being [~88m, Inf).
			Buckets:           metrics.ExponentialBuckets(0.01, 2, 20),
			StabilityLevel:    metrics.STABLE,
			DeprecatedVersion: "1.29.0",
		},
		[]string{"attempts"})

	DefinitionSchedulingSLIDuration = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Subsystem: SchedulerSubsystem,
			Name:      "definition_scheduling_sli_duration_seconds",
			Help:      "E2e latency for a definition being scheduled, from the time the definition enters the scheduling queue an d might involve multiple scheduling attempts.",
			// Start with 10ms with the last bucket being [~88m, Inf).
			Buckets:        metrics.ExponentialBuckets(0.01, 2, 20),
			StabilityLevel: metrics.BETA,
		},
		[]string{"attempts"})

	DefinitionSchedulingAttempts = metrics.NewHistogram(
		&metrics.HistogramOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "definition_scheduling_attempts",
			Help:           "Number of attempts to successfully schedule a definition.",
			Buckets:        metrics.ExponentialBuckets(1, 2, 5),
			StabilityLevel: metrics.STABLE,
		})

	FrameworkExtensionPointDuration = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Subsystem: SchedulerSubsystem,
			Name:      "framework_extension_point_duration_seconds",
			Help:      "Latency for running all plugins of a specific extension point.",
			// Start with 0.1ms with the last bucket being [~200ms, Inf)
			Buckets:        metrics.ExponentialBuckets(0.0001, 2, 12),
			StabilityLevel: metrics.STABLE,
		},
		[]string{"extension_point", "status", "profile"})

	PluginExecutionDuration = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Subsystem: SchedulerSubsystem,
			Name:      "plugin_execution_duration_seconds",
			Help:      "Duration for running a plugin at a specific extension point.",
			// Start with 0.01ms with the last bucket being [~22ms, Inf). We use a small factor (1.5)
			// so that we have better granularity since plugin latency is very sensitive.
			Buckets:        metrics.ExponentialBuckets(0.00001, 1.5, 20),
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"plugin", "extension_point", "status"})

	SchedulerQueueIncomingDefinitions = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "queue_incoming_definitions_total",
			Help:           "Number of definitions added to scheduling queues by event and queue type.",
			StabilityLevel: metrics.STABLE,
		}, []string{"queue", "event"})

	PermitWaitDuration = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "permit_wait_duration_seconds",
			Help:           "Duration of waiting on permit.",
			Buckets:        metrics.ExponentialBuckets(0.001, 2, 15),
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"result"})

	CacheSize = metrics.NewGaugeVec(
		&metrics.GaugeOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "scheduler_cache_size",
			Help:           "Number of nodes, definitions, and assumed (bound) definitions in the scheduler cache.",
			StabilityLevel: metrics.ALPHA,
		}, []string{"type"})

	unschedulableReasons = metrics.NewGaugeVec(
		&metrics.GaugeOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "unschedulable_definitions",
			Help:           "The number of unschedulable definitions broken down by plugin name. A definition will increment the gauge for all plugins that caused it to not schedule and so this metric have meaning only when broken down by plugin.",
			StabilityLevel: metrics.ALPHA,
		}, []string{"plugin", "profile"})

	PluginEvaluationTotal = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Subsystem:      SchedulerSubsystem,
			Name:           "plugin_evaluation_total",
			Help:           "Number of attempts to schedule definitions by each plugin and the extension point (available only in PreFilter, Filter, PreScore, and Score).",
			StabilityLevel: metrics.ALPHA,
		}, []string{"plugin", "extension_point", "profile"})

	metricsList = []metrics.Registerable{
		scheduleAttempts,
		schedulingLatency,
		SchedulingAlgorithmLatency,
		PreemptionVictims,
		PreemptionAttempts,
		pendingDefinitions,
		DefinitionSchedulingDuration,
		DefinitionSchedulingSLIDuration,
		DefinitionSchedulingAttempts,
		FrameworkExtensionPointDuration,
		PluginExecutionDuration,
		SchedulerQueueIncomingDefinitions,
		Goroutines,
		PermitWaitDuration,
		CacheSize,
		unschedulableReasons,
		PluginEvaluationTotal,
	}
)

var registerMetrics sync.Once

// Register all metrics.
func Register() {
	// Register the metrics.
	registerMetrics.Do(func() {
		RegisterMetrics(metricsList...)
	})
}

// RegisterMetrics registers a list of metrics.
// This function is exported because it is intended to be used by out-of-tree plugins to register their custom metrics.
func RegisterMetrics(extraMetrics ...metrics.Registerable) {
	for _, metric := range extraMetrics {
		legacyregistry.MustRegister(metric)
	}
}

// GetGather returns the gatherer. It used by test case outside current package.
func GetGather() metrics.Gatherer {
	return legacyregistry.DefaultGatherer
}

// ActiveDefinitions returns the pending definitions metrics with the label active
func ActiveDefinitions() metrics.GaugeMetric {
	return pendingDefinitions.With(metrics.Labels{"queue": "active"})
}

// BackoffDefinitions returns the pending definitions metrics with the label backoff
func BackoffDefinitions() metrics.GaugeMetric {
	return pendingDefinitions.With(metrics.Labels{"queue": "backoff"})
}

// UnschedulableDefinitions returns the pending definitions metrics with the label unschedulable
func UnschedulableDefinitions() metrics.GaugeMetric {
	return pendingDefinitions.With(metrics.Labels{"queue": "unschedulable"})
}

// GatedDefinitions returns the pending definitions metrics with the label gated
func GatedDefinitions() metrics.GaugeMetric {
	return pendingDefinitions.With(metrics.Labels{"queue": "gated"})
}

// SinceInSeconds gets the time since the specified start in seconds.
func SinceInSeconds(start time.Time) float64 {
	return time.Since(start).Seconds()
}

func UnschedulableReason(plugin string, profile string) metrics.GaugeMetric {
	return unschedulableReasons.With(metrics.Labels{"plugin": plugin, "profile": profile})
}
