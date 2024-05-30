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
	"time"

	"k8s.io/component-base/metrics"
)

// MetricRecorder represents a metric recorder which takes action when the
// metric Inc(), Dec() and Clear()
type MetricRecorder interface {
	Inc()
	Dec()
	Clear()
}

var _ MetricRecorder = &PendingDefinitionsRecorder{}

// PendingDefinitionsRecorder is an implementation of MetricRecorder
type PendingDefinitionsRecorder struct {
	recorder metrics.GaugeMetric
}

// NewActiveDefinitionsRecorder returns ActiveDefinitions in a Prometheus metric fashion
func NewActiveDefinitionsRecorder() *PendingDefinitionsRecorder {
	return &PendingDefinitionsRecorder{
		recorder: ActiveDefinitions(),
	}
}

// NewUnschedulableDefinitionsRecorder returns UnschedulableDefinitions in a Prometheus metric fashion
func NewUnschedulableDefinitionsRecorder() *PendingDefinitionsRecorder {
	return &PendingDefinitionsRecorder{
		recorder: UnschedulableDefinitions(),
	}
}

// NewBackoffDefinitionsRecorder returns BackoffDefinitions in a Prometheus metric fashion
func NewBackoffDefinitionsRecorder() *PendingDefinitionsRecorder {
	return &PendingDefinitionsRecorder{
		recorder: BackoffDefinitions(),
	}
}

// NewGatedDefinitionsRecorder returns GatedDefinitions in a Prometheus metric fashion
func NewGatedDefinitionsRecorder() *PendingDefinitionsRecorder {
	return &PendingDefinitionsRecorder{
		recorder: GatedDefinitions(),
	}
}

// Inc increases a metric counter by 1, in an atomic way
func (r *PendingDefinitionsRecorder) Inc() {
	r.recorder.Inc()
}

// Dec decreases a metric counter by 1, in an atomic way
func (r *PendingDefinitionsRecorder) Dec() {
	r.recorder.Dec()
}

// Clear set a metric counter to 0, in an atomic way
func (r *PendingDefinitionsRecorder) Clear() {
	r.recorder.Set(float64(0))
}

// metric is the data structure passed in the buffer channel between the main framework thread
// and the metricsRecorder goroutine.
type metric struct {
	metric      *metrics.HistogramVec
	labelValues []string
	value       float64
}

// MetricAsyncRecorder records metric in a separate goroutine to avoid overhead in the critical path.
type MetricAsyncRecorder struct {
	// bufferCh is a channel that serves as a metrics buffer before the metricsRecorder goroutine reports it.
	bufferCh chan *metric
	// if bufferSize is reached, incoming metrics will be discarded.
	bufferSize int
	// how often the recorder runs to flush the metrics.
	interval time.Duration

	// stopCh is used to stop the goroutine which periodically flushes metrics.
	stopCh <-chan struct{}
	// IsStoppedCh indicates whether the goroutine is stopped. It's used in tests only to make sure
	// the metric flushing goroutine is stopped so that tests can collect metrics for verification.
	IsStoppedCh chan struct{}
}

func NewMetricsAsyncRecorder(bufferSize int, interval time.Duration, stopCh <-chan struct{}) *MetricAsyncRecorder {
	recorder := &MetricAsyncRecorder{
		bufferCh:    make(chan *metric, bufferSize),
		bufferSize:  bufferSize,
		interval:    interval,
		stopCh:      stopCh,
		IsStoppedCh: make(chan struct{}),
	}
	go recorder.run()
	return recorder
}

// ObservePluginDurationAsync observes the plugin_execution_duration_seconds metric.
// The metric will be flushed to Prometheus asynchronously.
func (r *MetricAsyncRecorder) ObservePluginDurationAsync(extensionPoint, pluginName, status string, value float64) {
	newMetric := &metric{
		metric:      PluginExecutionDuration,
		labelValues: []string{pluginName, extensionPoint, status},
		value:       value,
	}
	select {
	case r.bufferCh <- newMetric:
	default:
	}
}

// run flushes buffered metrics into Prometheus every second.
func (r *MetricAsyncRecorder) run() {
	for {
		select {
		case <-r.stopCh:
			close(r.IsStoppedCh)
			return
		default:
		}
		r.FlushMetrics()
		time.Sleep(r.interval)
	}
}

// FlushMetrics tries to clean up the bufferCh by reading at most bufferSize metrics.
func (r *MetricAsyncRecorder) FlushMetrics() {
	for i := 0; i < r.bufferSize; i++ {
		select {
		case m := <-r.bufferCh:
			m.metric.WithLabelValues(m.labelValues...).Observe(m.value)
		default:
			return
		}
	}
}
