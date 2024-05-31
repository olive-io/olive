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

const (
	LabelHostname = "olive.io/hostname"

	// Label value is the network location of kube-apiserver stored as <ip:port>
	// Stored in APIServer Identity lease objects to view what address is used for peer proxy
	AnnotationPeerAdvertiseAddress = "olive.io/peer-advertise-address"

	LabelTopologyZone   = "topology.olive.io/zone"
	LabelTopologyRegion = "topology.olive.io/region"

	// These label have been deprecated since 1.17, but will be supported for
	// the foreseeable future, to accommodate things like long-lived PVs that
	// use them.  New users should prefer the "topology.olive.io/*"
	// equivalents.
	LabelFailureDomainBetaZone   = "failure-domain.beta.olive.io/zone"   // deprecated
	LabelFailureDomainBetaRegion = "failure-domain.beta.olive.io/region" // deprecated

	// Retained for compat when vendored.  Do not use these consts in new code.
	LabelZoneFailureDomain       = LabelFailureDomainBetaZone   // deprecated
	LabelZoneRegion              = LabelFailureDomainBetaRegion // deprecated
	LabelZoneFailureDomainStable = LabelTopologyZone            // deprecated
	LabelZoneRegionStable        = LabelTopologyRegion          // deprecated

	LabelInstanceType       = "beta.olive.io/instance-type"
	LabelInstanceTypeStable = "node.olive.io/instance-type"

	LabelOSStable   = "olive.io/os"
	LabelArchStable = "olive.io/arch"

	// LabelWindowsBuild is used on Windows nodes to specify the Windows build number starting with v1.17.0.
	// It's in the format MajorVersion.MinorVersion.BuildNumber (for ex: 10.0.17763)
	LabelWindowsBuild = "node.olive.io/windows-build"

	// LabelNamespaceSuffixKubelet is an allowed label namespace suffix kubelets can self-set ([*.]kubelet.olive.io/*)
	LabelNamespaceSuffixKubelet = "kubelet.olive.io"
	// LabelNamespaceSuffixNode is an allowed label namespace suffix kubelets can self-set ([*.]node.olive.io/*)
	LabelNamespaceSuffixNode = "node.olive.io"

	// LabelNamespaceNodeRestriction is a forbidden label namespace that kubelets may not self-set when the NodeRestriction admission plugin is enabled
	LabelNamespaceNodeRestriction = "node-restriction.olive.io"

	// IsHeadlessService is added by Controller to an Endpoint denoting if its parent
	// Service is Headless. The existence of this label can be used further by other
	// controllers and kube-proxy to check if the Endpoint objects should be replicated when
	// using Headless Services
	IsHeadlessService = "service.olive.io/headless"

	// LabelNodeExcludeBalancers specifies that the node should not be considered as a target
	// for external load-balancers which use nodes as a second hop (e.g. many cloud LBs which only
	// understand nodes). For services that use externalTrafficPolicy=Local, this may mean that
	// any backends on excluded nodes are not reachable by those external load-balancers.
	// Implementations of this exclusion may vary based on provider.
	LabelNodeExcludeBalancers = "node.olive.io/exclude-from-external-load-balancers"
	// LabelMetadataName is the label name which, in-tree, is used to automatically label namespaces, so they can be selected easily by tools which require definitive labels
	LabelMetadataName = "olive.io/metadata.name"
)
