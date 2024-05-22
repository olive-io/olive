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

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/olive-io/olive/client/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Definitions returns a DefinitionInformer.
	Definitions() DefinitionInformer
	// DefinitionLists returns a DefinitionListInformer.
	DefinitionLists() DefinitionListInformer
	// ProcessInstances returns a ProcessInstanceInformer.
	ProcessInstances() ProcessInstanceInformer
	// ProcessInstanceLists returns a ProcessInstanceListInformer.
	ProcessInstanceLists() ProcessInstanceListInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Definitions returns a DefinitionInformer.
func (v *version) Definitions() DefinitionInformer {
	return &definitionInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// DefinitionLists returns a DefinitionListInformer.
func (v *version) DefinitionLists() DefinitionListInformer {
	return &definitionListInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ProcessInstances returns a ProcessInstanceInformer.
func (v *version) ProcessInstances() ProcessInstanceInformer {
	return &processInstanceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ProcessInstanceLists returns a ProcessInstanceListInformer.
func (v *version) ProcessInstanceLists() ProcessInstanceListInformer {
	return &processInstanceListInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
