package meta

import (
	goruntime "runtime"

	"github.com/olive-io/olive/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
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
		"meta_version": version.GoV(),
	}).Set(1)
	currentGoVersion.With(prometheus.Labels{
		"meta_go_version": goruntime.Version(),
	}).Set(1)
}
