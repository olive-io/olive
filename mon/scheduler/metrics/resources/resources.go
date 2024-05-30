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

// Package resources provides a metrics collector that reports the
// resource consumption (requests and limits) of the definitions in the cluster
// as the scheduler and olive-runner would interpret it.
package resources

import (
	"net/http"
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/component-base/metrics"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	corelisters "github.com/olive-io/olive/client/generated/listers/core/v1"
	v1resource "github.com/olive-io/olive/pkg/api/v1/resource"
)

type resourceLifecycleDescriptors struct {
	total *metrics.Desc
}

func (d resourceLifecycleDescriptors) Describe(ch chan<- *metrics.Desc) {
	ch <- d.total
}

type resourceMetricsDescriptors struct {
	requests resourceLifecycleDescriptors
	limits   resourceLifecycleDescriptors
}

func (d resourceMetricsDescriptors) Describe(ch chan<- *metrics.Desc) {
	d.requests.Describe(ch)
	d.limits.Describe(ch)
}

var definitionResourceDesc = resourceMetricsDescriptors{
	requests: resourceLifecycleDescriptors{
		total: metrics.NewDesc("olive_definition_resource_request",
			"Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.",
			[]string{"namespace", "definition", "runner", "scheduler", "priority", "resource", "unit"},
			nil,
			metrics.STABLE,
			""),
	},
	limits: resourceLifecycleDescriptors{
		total: metrics.NewDesc("olive_definition_resource_limit",
			"Resources limit for workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.",
			[]string{"namespace", "definition", "runner", "scheduler", "priority", "resource", "unit"},
			nil,
			metrics.STABLE,
			""),
	},
}

// Handler creates a collector from the provided definitionLister and returns an http.Handler that
// will report the requested metrics in the prometheus format. It does not include any other
// metrics.
func Handler(definitionLister corelisters.DefinitionLister) http.Handler {
	collector := NewDefinitionResourcesMetricsCollector(definitionLister)
	registry := metrics.NewKubeRegistry()
	registry.CustomMustRegister(collector)
	return metrics.HandlerWithReset(registry, metrics.HandlerOpts{})
}

// Check if resourceMetricsCollector implements necessary interface
var _ metrics.StableCollector = &definitionResourceCollector{}

// NewDefinitionResourcesMetricsCollector registers a O(definitions) cardinality metric that
// reports the current resources requested by all definitions on the cluster within
// the Kubernetes resource model. Metrics are broken down by definition, runner, resource,
// and phase of lifecycle. Each definition returns two series per resource - one for
// their aggregate usage (required to schedule) and one for their phase specific
// usage. This allows admins to assess the cost per resource at different phases
// of startup and compare to actual resource usage.
func NewDefinitionResourcesMetricsCollector(definitionLister corelisters.DefinitionLister) metrics.StableCollector {
	return &definitionResourceCollector{
		lister: definitionLister,
	}
}

type definitionResourceCollector struct {
	metrics.BaseStableCollector
	lister corelisters.DefinitionLister
}

func (c *definitionResourceCollector) DescribeWithStability(ch chan<- *metrics.Desc) {
	definitionResourceDesc.Describe(ch)
}

func (c *definitionResourceCollector) CollectWithStability(ch chan<- metrics.Metric) {
	definitions, err := c.lister.List(labels.Everything())
	if err != nil {
		return
	}
	reuseReqs, reuseLimits := make(corev1.ResourceList, 4), make(corev1.ResourceList, 4)
	for _, p := range definitions {
		reqs, limits, terminal := definitionRequestsAndLimitsByLifecycle(p, reuseReqs, reuseLimits)
		if terminal {
			// terminal definitions are excluded from resource usage calculations
			continue
		}
		for _, t := range []struct {
			desc  resourceLifecycleDescriptors
			total corev1.ResourceList
		}{
			{
				desc:  definitionResourceDesc.requests,
				total: reqs,
			},
			{
				desc:  definitionResourceDesc.limits,
				total: limits,
			},
		} {
			for resourceName, val := range t.total {
				var unitName string
				switch resourceName {
				case corev1.ResourceCPU:
					unitName = "cores"
				case corev1.ResourceMemory:
					unitName = "bytes"
				case corev1.ResourceStorage:
					unitName = "bytes"
				case corev1.ResourceEphemeralStorage:
					unitName = "bytes"
				default:
				}
				var priority string
				if p.Spec.Priority != nil {
					priority = strconv.FormatInt(*p.Spec.Priority, 10)
				}
				recordMetricWithUnit(ch, t.desc.total, p.Namespace, p.Name, p.Spec.RegionName, priority, resourceName, unitName, val)
			}
		}
	}
}

func recordMetricWithUnit(
	ch chan<- metrics.Metric,
	desc *metrics.Desc,
	namespace, name, runnerName, priority string,
	resourceName corev1.ResourceName,
	unit string,
	val resource.Quantity,
) {
	if val.IsZero() {
		return
	}
	ch <- metrics.NewLazyConstMetric(desc, metrics.GaugeValue,
		val.AsApproximateFloat64(),
		namespace, name, runnerName, priority, string(resourceName), unit,
	)
}

// definitionRequestsAndLimitsByLifecycle returns a dictionary of all defined resources summed up for all
// containers of the definition. Definition overhead is added to the
// total container resource requests and to the total container limits which have a
// non-zero quantity. The caller may avoid allocations of resource lists by passing
// a requests and limits list to the function, which will be cleared before use.
// This method is the same as v1resource.DefinitionRequestsAndLimits but avoids allocating in several
// scenarios for efficiency.
func definitionRequestsAndLimitsByLifecycle(def *corev1.Definition, reuseReqs, reuseLimits corev1.ResourceList) (reqs, limits corev1.ResourceList, terminal bool) {
	switch {
	case len(def.Spec.RegionName) == 0:
		// unscheduled definitions cannot be terminal
	case def.Status.Phase == corev1.DefSucceeded, def.Status.Phase == corev1.DefFailed:
		terminal = true
	}
	if terminal {
		return
	}

	reqs = v1resource.DefinitionRequests(def, v1resource.DefinitionResourcesOptions{Reuse: reuseReqs})
	limits = v1resource.DefinitionLimits(def, v1resource.DefinitionResourcesOptions{Reuse: reuseLimits})
	return
}
