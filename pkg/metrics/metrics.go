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

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/atomic"
)

type Gauge interface {
	prometheus.Gauge
	Get() float64
}

type gaugeFunc struct {
	prometheus.GaugeFunc
	value *atomic.Float64
}

func NewGauge(opts prometheus.GaugeOpts) Gauge {
	gf := gaugeFunc{
		value: atomic.NewFloat64(0),
	}
	fn := prometheus.NewGaugeFunc(opts, func() float64 {
		return gf.value.Load()
	})
	gf.GaugeFunc = fn

	return &gf
}

func (g *gaugeFunc) Set(v float64) {
	g.value.Store(v)
}

func (g *gaugeFunc) Inc() {
	g.value.Add(1)
}

func (g *gaugeFunc) Dec() {
	g.value.Sub(1)
}

func (g *gaugeFunc) Add(v float64) {
	g.value.Add(v)
}

func (g *gaugeFunc) Sub(v float64) {
	g.value.Sub(v)
}

func (g *gaugeFunc) SetToCurrentTime() {
	g.Set(float64(time.Now().UnixNano()) / 1e9)
}

func (g *gaugeFunc) Get() float64 {
	return g.value.Load()
}
