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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package applyconfiguration

import (
	v1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	apidiscoveryv1 "github.com/olive-io/olive/client-go/generated/applyconfiguration/apidiscovery/v1"
	applyconfigurationcorev1 "github.com/olive-io/olive/client-go/generated/applyconfiguration/core/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

// ForKind returns an apply configuration type for the given GroupVersionKind, or nil if no
// apply configuration type exists for the given GroupVersionKind.
func ForKind(kind schema.GroupVersionKind) interface{} {
	switch kind {
	// Group=apidiscovery.olive.io, Version=v1
	case v1.SchemeGroupVersion.WithKind("Box"):
		return &apidiscoveryv1.BoxApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("Edge"):
		return &apidiscoveryv1.EdgeApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("EdgeSpec"):
		return &apidiscoveryv1.EdgeSpecApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("Endpoint"):
		return &apidiscoveryv1.EndpointApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("EndpointSpec"):
		return &apidiscoveryv1.EndpointSpecApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("Node"):
		return &apidiscoveryv1.NodeApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("Service"):
		return &apidiscoveryv1.ServiceApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("ServiceSpec"):
		return &apidiscoveryv1.ServiceSpecApplyConfiguration{}

		// Group=core.olive.io, Version=v1
	case corev1.SchemeGroupVersion.WithKind("BpmnStat"):
		return &applyconfigurationcorev1.BpmnStatApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("Definition"):
		return &applyconfigurationcorev1.DefinitionApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("DefinitionSpec"):
		return &applyconfigurationcorev1.DefinitionSpecApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("DefinitionStatus"):
		return &applyconfigurationcorev1.DefinitionStatusApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("Namespace"):
		return &applyconfigurationcorev1.NamespaceApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("NamespaceCondition"):
		return &applyconfigurationcorev1.NamespaceConditionApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("NamespaceSpec"):
		return &applyconfigurationcorev1.NamespaceSpecApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("NamespaceStatus"):
		return &applyconfigurationcorev1.NamespaceStatusApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("Process"):
		return &applyconfigurationcorev1.ProcessApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("ProcessSpec"):
		return &applyconfigurationcorev1.ProcessSpecApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("ProcessStatus"):
		return &applyconfigurationcorev1.ProcessStatusApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("Region"):
		return &applyconfigurationcorev1.RegionApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("RegionReplica"):
		return &applyconfigurationcorev1.RegionReplicaApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("RegionSpec"):
		return &applyconfigurationcorev1.RegionSpecApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("RegionStat"):
		return &applyconfigurationcorev1.RegionStatApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("RegionStatus"):
		return &applyconfigurationcorev1.RegionStatusApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("Runner"):
		return &applyconfigurationcorev1.RunnerApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("RunnerSpec"):
		return &applyconfigurationcorev1.RunnerSpecApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("RunnerStat"):
		return &applyconfigurationcorev1.RunnerStatApplyConfiguration{}
	case corev1.SchemeGroupVersion.WithKind("RunnerStatus"):
		return &applyconfigurationcorev1.RunnerStatusApplyConfiguration{}

	}
	return nil
}
