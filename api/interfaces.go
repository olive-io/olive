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

package api

// ObjectKind all objects that are serialized from a Scheme encode their type information. This interface is used
// by serialization set type information from the Scheme onto the serialized version of an object.
// For objects that cannot be serialized or have unique information, this interface may be a no-op
type ObjectKind interface {
	// SetGroupVersionKind returns the stored group, version, and kind of an object, or an empty struct
	// should clear the current setting.
	SetGroupVersionKind(kind GroupVersionKind)
	// GroupVersionKind returns the stored group, version, and kind of an object, or an empty struct
	// if the object does not expose or provide these fields.
	GroupVersionKind() GroupVersionKind
}

type MetaInterface interface {
	GetName() string
	SetName(name string)
	GetUID() string
	SetUID(uid string)
	GetDescription() string
	SetDescription(description string)
	GetNamespace() string
	SetNamespace(namespace string)
	GetCreationTimestamp() int64
	SetCreationTimestamp(timestamp int64)
	GetUpdateTimestamp() int64
	SetUpdateTimestamp(timestamp int64)
	GetDeletionTimestamp() int64
	SetDeletionTimestamp(timestamp int64)
	GetResourceVersion() int64
	SetResourceVersion(rev int64)

	GetLabels() map[string]string
	SetLabels(labels map[string]string)
	GetAnnotations() map[string]string
	SetAnnotations(annotations map[string]string)
}

type ListInterface interface {
	GetLimit() int64
	SetLimit(limit int64)
	GetContinue() string
	SetContinue(token string)
	GetTotal() int64
	SetTotal(total int64)
}

type Object interface {
	GetObjectKind() ObjectKind

	DeepCopyObject() Object
}
