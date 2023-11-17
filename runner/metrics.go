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
		"runner_version": version.Version,
	}).Set(1)
	currentGoVersion.With(prometheus.Labels{
		"runner_go_version": goruntime.Version(),
	}).Set(1)
}
