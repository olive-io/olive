// Copyright 2016 The olive Authors
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

package backend

import "github.com/prometheus/client_golang/prometheus"

var (
	commitSec = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "olive",
		Subsystem: "disk",
		Name:      "backend_commit_duration_seconds",
		Help:      "The latency distributions of commit called by backend.",

		// lowest bucket start of upper bound 0.001 sec (1 ms) with factor 2
		// highest bucket start of 0.001 sec * 2^13 == 8.192 sec
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
	})

	writeSec = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "olive_debugging",
		Subsystem: "disk",
		Name:      "backend_commit_write_duration_seconds",
		Help:      "The latency distributions of commit.write called by pebble backend.",

		// lowest bucket start of upper bound 0.001 sec (1 ms) with factor 2
		// highest bucket start of 0.001 sec * 2^13 == 8.192 sec
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
	})

	snapshotTransferSec = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "olive",
		Subsystem: "disk",
		Name:      "backend_snapshot_duration_seconds",
		Help:      "The latency distribution of backend snapshots.",

		// lowest bucket start of upper bound 0.01 sec (10 ms) with factor 2
		// highest bucket start of 0.01 sec * 2^16 == 655.36 sec
		Buckets: prometheus.ExponentialBuckets(.01, 2, 17),
	})
)

func init() {
	prometheus.MustRegister(commitSec)
	prometheus.MustRegister(writeSec)
	prometheus.MustRegister(snapshotTransferSec)
}
