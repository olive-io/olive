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

import (
	"k8s.io/apimachinery/pkg/labels"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

var (
// rcKind = v1.SchemeGroupVersion.WithKind("ReplicationController")
)

// DefaultSelector returns a selector deduced from the Services, Replication
// Controllers, Replica Sets, and Stateful Sets matching the given region.
func DefaultSelector(
	region *corev1.Region,
	sl corelisters.ServiceLister,
	cl corelisters.ReplicationControllerLister,
	rsl appslisters.ReplicaSetLister,
	ssl appslisters.StatefulSetLister,
) labels.Selector {
	labelSet := make(labels.Set)
	// Since services, RCs, RSs and SSs match the region, they won't have conflicting
	// labels. Merging is safe.

	//if services, err := GetRegionServices(sl, region); err == nil {
	//	for _, service := range services {
	//		labelSet = labels.Merge(labelSet, service.Spec.Selector)
	//	}
	//}
	selector := labelSet.AsSelector()
	//
	//owner := metav1.GetControllerOfNoCopy(region)
	//if owner == nil {
	//	return selector
	//}
	//
	//gv, err := schema.ParseGroupVersion(owner.APIVersion)
	//if err != nil {
	//	return selector
	//}
	//
	//gvk := gv.WithKind(owner.Kind)
	//switch gvk {
	//case rcKind:
	//	if rc, err := cl.ReplicationControllers(region.Namespace).Get(owner.Name); err == nil {
	//		labelSet = labels.Merge(labelSet, rc.Spec.Selector)
	//		selector = labelSet.AsSelector()
	//	}
	//default:
	//	// Not owned by a supported controller.
	//}

	return selector
}

// GetRegionServices gets the services that have the selector that match the labels on the given region.
//func GetRegionServices(sl corelisters.ServiceLister, region *corev1.Region) ([]*v1.Service, error) {
//	allServices, err := sl.Services(region.Namespace).List(labels.Everything())
//	if err != nil {
//		return nil, err
//	}
//
//	var services []*v1.Service
//	for i := range allServices {
//		service := allServices[i]
//		if service.Spec.Selector == nil {
//			// services with nil selectors match nothing, not everything.
//			continue
//		}
//		selector := labels.Set(service.Spec.Selector).AsSelectorPreValidated()
//		if selector.Matches(labels.Set(region.Labels)) {
//			services = append(services, service)
//		}
//	}
//
//	return services, nil
//}
