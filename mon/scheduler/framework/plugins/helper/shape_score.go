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

package helper

// FunctionShape represents a collection of FunctionShapePoint.
type FunctionShape []FunctionShapePoint

// FunctionShapePoint represents a shape point.
type FunctionShapePoint struct {
	// Utilization is function argument.
	Utilization int64
	// Score is function value.
	Score int64
}

// BuildBrokenLinearFunction creates a function which is built using linear segments. Segments are defined via shape array.
// Shape[i].Utilization slice represents points on "Utilization" axis where different segments meet.
// Shape[i].Score represents function values at meeting points.
//
// function f(p) is defined as:
//
//	shape[0].Score for p < shape[0].Utilization
//	shape[n-1].Score for p > shape[n-1].Utilization
//
// and linear between points (p < shape[i].Utilization)
func BuildBrokenLinearFunction(shape FunctionShape) func(int64) int64 {
	return func(p int64) int64 {
		for i := 0; i < len(shape); i++ {
			if p <= int64(shape[i].Utilization) {
				if i == 0 {
					return shape[0].Score
				}
				return shape[i-1].Score + (shape[i].Score-shape[i-1].Score)*(p-shape[i-1].Utilization)/(shape[i].Utilization-shape[i-1].Utilization)
			}
		}
		return shape[len(shape)-1].Score
	}
}
