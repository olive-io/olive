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

package scheduler

import corev1 "github.com/olive-io/olive/apis/core/v1"

// scheduler inner message channel
type imessage interface {
	payload() any
}

type allocRegionBox struct {
	num int
}

// allocRegionMsg defines create new Region
type allocRegionMsg struct {
	box *allocRegionBox
}

func newAllocRegionMsg(n int) *allocRegionMsg {
	return &allocRegionMsg{box: &allocRegionBox{num: n}}
}

func (m *allocRegionMsg) payload() any {
	return m.box
}

type scaleRegionBox struct {
	region *corev1.Region
	scale  int
}

type scaleRegionMsg struct {
	box *scaleRegionBox
}

func newScaleRegionMsg(region *corev1.Region, n int) *scaleRegionMsg {
	return &scaleRegionMsg{box: &scaleRegionBox{region: region, scale: n}}
}

func (m *scaleRegionMsg) payload() any {
	return m.box
}
