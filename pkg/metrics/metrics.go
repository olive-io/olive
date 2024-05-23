/*
Copyright 2023 The olive Authors

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
