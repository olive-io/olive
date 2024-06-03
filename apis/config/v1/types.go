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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EtcdCluster struct {
	metav1.TypeMeta `json:",inline"`

	Endpoints []string `json:"endpoints" protobuf:"bytes,1,rep,name=endpoints"`
	// dial and request timeout
	Timeout         string `json:"timeout" protobuf:"bytes,2,opt,name=timeout"`
	MaxUnaryRetries int32  `json:"maxUnaryRetries" protobuf:"varint,3,opt,name=maxUnaryRetries"`
}
