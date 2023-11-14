package membership

import "github.com/prometheus/client_golang/prometheus"

var (
	ClusterVersionMetrics = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "olive",
		Subsystem: "cluster",
		Name:      "version",
		Help:      "Which version is running. 1 for 'cluster_version' label with current cluster version",
	},
		[]string{"cluster_version"})
)

func init() {
	prometheus.MustRegister(ClusterVersionMetrics)
}
