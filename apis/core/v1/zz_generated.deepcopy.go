//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1

import (
	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BpmnStat) DeepCopyInto(out *BpmnStat) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BpmnStat.
func (in *BpmnStat) DeepCopy() *BpmnStat {
	if in == nil {
		return nil
	}
	out := new(BpmnStat)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Definition) DeepCopyInto(out *Definition) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Definition.
func (in *Definition) DeepCopy() *Definition {
	if in == nil {
		return nil
	}
	out := new(Definition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Definition) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DefinitionList) DeepCopyInto(out *DefinitionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Definition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DefinitionList.
func (in *DefinitionList) DeepCopy() *DefinitionList {
	if in == nil {
		return nil
	}
	out := new(DefinitionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DefinitionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DefinitionSpec) DeepCopyInto(out *DefinitionSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DefinitionSpec.
func (in *DefinitionSpec) DeepCopy() *DefinitionSpec {
	if in == nil {
		return nil
	}
	out := new(DefinitionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DefinitionStatus) DeepCopyInto(out *DefinitionStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DefinitionStatus.
func (in *DefinitionStatus) DeepCopy() *DefinitionStatus {
	if in == nil {
		return nil
	}
	out := new(DefinitionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlowNodeStat) DeepCopyInto(out *FlowNodeStat) {
	*out = *in
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.DataObjects != nil {
		in, out := &in.DataObjects, &out.DataObjects
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlowNodeStat.
func (in *FlowNodeStat) DeepCopy() *FlowNodeStat {
	if in == nil {
		return nil
	}
	out := new(FlowNodeStat)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Namespace) DeepCopyInto(out *Namespace) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Namespace.
func (in *Namespace) DeepCopy() *Namespace {
	if in == nil {
		return nil
	}
	out := new(Namespace)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Namespace) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespaceCondition) DeepCopyInto(out *NamespaceCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespaceCondition.
func (in *NamespaceCondition) DeepCopy() *NamespaceCondition {
	if in == nil {
		return nil
	}
	out := new(NamespaceCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespaceList) DeepCopyInto(out *NamespaceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Namespace, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespaceList.
func (in *NamespaceList) DeepCopy() *NamespaceList {
	if in == nil {
		return nil
	}
	out := new(NamespaceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NamespaceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespaceSpec) DeepCopyInto(out *NamespaceSpec) {
	*out = *in
	if in.Finalizers != nil {
		in, out := &in.Finalizers, &out.Finalizers
		*out = make([]FinalizerName, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespaceSpec.
func (in *NamespaceSpec) DeepCopy() *NamespaceSpec {
	if in == nil {
		return nil
	}
	out := new(NamespaceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespaceStatus) DeepCopyInto(out *NamespaceStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]NamespaceCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespaceStatus.
func (in *NamespaceStatus) DeepCopy() *NamespaceStatus {
	if in == nil {
		return nil
	}
	out := new(NamespaceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Process) DeepCopyInto(out *Process) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Process.
func (in *Process) DeepCopy() *Process {
	if in == nil {
		return nil
	}
	out := new(Process)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Process) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProcessList) DeepCopyInto(out *ProcessList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Process, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessList.
func (in *ProcessList) DeepCopy() *ProcessList {
	if in == nil {
		return nil
	}
	out := new(ProcessList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProcessList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProcessRunningState) DeepCopyInto(out *ProcessRunningState) {
	*out = *in
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.DataObjects != nil {
		in, out := &in.DataObjects, &out.DataObjects
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Variables != nil {
		in, out := &in.Variables, &out.Variables
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessRunningState.
func (in *ProcessRunningState) DeepCopy() *ProcessRunningState {
	if in == nil {
		return nil
	}
	out := new(ProcessRunningState)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProcessSpec) DeepCopyInto(out *ProcessSpec) {
	*out = *in
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = make(map[string]apidiscoveryv1.Box, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.DataObjects != nil {
		in, out := &in.DataObjects, &out.DataObjects
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessSpec.
func (in *ProcessSpec) DeepCopy() *ProcessSpec {
	if in == nil {
		return nil
	}
	out := new(ProcessSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProcessStat) DeepCopyInto(out *ProcessStat) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.State.DeepCopyInto(&out.State)
	if in.FlowNodes != nil {
		in, out := &in.FlowNodes, &out.FlowNodes
		*out = make(map[string]FlowNodeStat, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessStat.
func (in *ProcessStat) DeepCopy() *ProcessStat {
	if in == nil {
		return nil
	}
	out := new(ProcessStat)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProcessStat) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProcessStatus) DeepCopyInto(out *ProcessStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessStatus.
func (in *ProcessStatus) DeepCopy() *ProcessStatus {
	if in == nil {
		return nil
	}
	out := new(ProcessStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Region) DeepCopyInto(out *Region) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Region.
func (in *Region) DeepCopy() *Region {
	if in == nil {
		return nil
	}
	out := new(Region)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Region) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegionList) DeepCopyInto(out *RegionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Region, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegionList.
func (in *RegionList) DeepCopy() *RegionList {
	if in == nil {
		return nil
	}
	out := new(RegionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RegionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegionReplica) DeepCopyInto(out *RegionReplica) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegionReplica.
func (in *RegionReplica) DeepCopy() *RegionReplica {
	if in == nil {
		return nil
	}
	out := new(RegionReplica)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegionSpec) DeepCopyInto(out *RegionSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = make([]RegionReplica, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegionSpec.
func (in *RegionSpec) DeepCopy() *RegionSpec {
	if in == nil {
		return nil
	}
	out := new(RegionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegionStat) DeepCopyInto(out *RegionStat) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Bpmn != nil {
		in, out := &in.Bpmn, &out.Bpmn
		*out = new(BpmnStat)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegionStat.
func (in *RegionStat) DeepCopy() *RegionStat {
	if in == nil {
		return nil
	}
	out := new(RegionStat)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RegionStat) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegionStatus) DeepCopyInto(out *RegionStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegionStatus.
func (in *RegionStatus) DeepCopy() *RegionStatus {
	if in == nil {
		return nil
	}
	out := new(RegionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Runner) DeepCopyInto(out *Runner) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Runner.
func (in *Runner) DeepCopy() *Runner {
	if in == nil {
		return nil
	}
	out := new(Runner)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Runner) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RunnerList) DeepCopyInto(out *RunnerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Runner, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RunnerList.
func (in *RunnerList) DeepCopy() *RunnerList {
	if in == nil {
		return nil
	}
	out := new(RunnerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RunnerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RunnerSpec) DeepCopyInto(out *RunnerSpec) {
	*out = *in
	if in.IsLearner != nil {
		in, out := &in.IsLearner, &out.IsLearner
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RunnerSpec.
func (in *RunnerSpec) DeepCopy() *RunnerSpec {
	if in == nil {
		return nil
	}
	out := new(RunnerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RunnerStat) DeepCopyInto(out *RunnerStat) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Bpmn != nil {
		in, out := &in.Bpmn, &out.Bpmn
		*out = new(BpmnStat)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RunnerStat.
func (in *RunnerStat) DeepCopy() *RunnerStat {
	if in == nil {
		return nil
	}
	out := new(RunnerStat)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RunnerStat) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RunnerStatList) DeepCopyInto(out *RunnerStatList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Runner, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RunnerStatList.
func (in *RunnerStatList) DeepCopy() *RunnerStatList {
	if in == nil {
		return nil
	}
	out := new(RunnerStatList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RunnerStatList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RunnerStatus) DeepCopyInto(out *RunnerStatus) {
	*out = *in
	if in.Regions != nil {
		in, out := &in.Regions, &out.Regions
		*out = make([]int64, len(*in))
		copy(*out, *in)
	}
	if in.Leaders != nil {
		in, out := &in.Leaders, &out.Leaders
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RunnerStatus.
func (in *RunnerStatus) DeepCopy() *RunnerStatus {
	if in == nil {
		return nil
	}
	out := new(RunnerStatus)
	in.DeepCopyInto(out)
	return out
}
