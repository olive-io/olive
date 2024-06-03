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

package v1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

func addConversionFuncs(scheme *runtime.Scheme) error {
	err := scheme.AddFieldLabelConversionFunc(SchemeGroupVersion.WithKind("Runner"),
		func(label, value string) (string, string, error) {
			switch label {
			case "metadata.name",
				"metadata.namespace",
				"spec.id",
				"spec.name",
				"status.phase":
				return label, value, nil
			// This is for backwards compatibility with old v1 clients which send spec.host
			case "spec.peerURLs":
				return "spec.peers", value, nil
			case "spec.clientURLs":
				return "spec.clients", value, nil
			default:
				return "", "", fmt.Errorf("field label not supported: %s", label)
			}
		},
	)
	if err != nil {
		return err
	}
	err = scheme.AddFieldLabelConversionFunc(SchemeGroupVersion.WithKind("Region"),
		func(label, value string) (string, string, error) {
			switch label {
			case "metadata.name",
				"metadata.namespace",
				"spec.id",
				"spec.replicas",
				"status.phase":
				return label, value, nil
			default:
				return "", "", fmt.Errorf("field label not supported: %s", label)
			}
		},
	)
	if err != nil {
		return err
	}
	return err
}
