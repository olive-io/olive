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

// +k8s:deepcopy-gen=package
// +k8s:protobuf-gen=package
// +k8s:defaulter-gen=TypeMeta
// +k8s:defaulter-gen-input=github.com/olive-io/olive/apis/config/v1
// +k8s:conversion-gen=github.com/olive-io/olive/apis/config
// +k8s:openapi-gen=true
// +groupName=mon.olive.io

// Package v1 is the internal version of the API.
package v1 // import "github.com/olive-io/olive/apis/config/v1"
