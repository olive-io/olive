/*
Copyright 2025 The olive Authors

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

package types

import (
	"path"
)

func (in *Endpoint) URL() string {
	url := path.Join(in.Type.String())
	if in.Kind != "" {
		url = path.Join(url, in.Kind)
	}
	if in.Name != "" {
		url = path.Join(url, in.Name)
	}
	return url
}

func (in *ProcessContext) DeepCopyInto(out *ProcessContext) {
	*out = *in
	for k, v := range in.DataObjects {
		out.DataObjects[k] = v
	}
	for k, v := range in.Variables {
		out.Variables[k] = v
	}
}

func (in *ProcessContext) DeepCopy() *ProcessContext {
	if in == nil {
		return nil
	}
	out := new(ProcessContext)
	in.DeepCopyInto(out)
	return out
}

func (in *Process) DeepCopyInto(out *Process) {
	*out = *in
	if in.Context != nil {
		in, out := in.Context, out.Context
		in.DeepCopyInto(out)
	}
}

func (in *Process) DeepCopy() *Process {
	if in == nil {
		return nil
	}
	out := new(Process)
	in.DeepCopyInto(out)
	return out
}

func (in *FlowNode) DeepCopyInto(out *FlowNode) {
	*out = *in
	if in.Headers != nil {
		for k, v := range in.Headers {
			out.Headers[k] = v
		}
	}
	if in.Properties != nil {
		for k, v := range in.Properties {
			out.Properties[k] = v
		}
	}
	if in.DataObjects != nil {
		for k, v := range in.DataObjects {
			out.DataObjects[k] = v
		}
	}
}

func (in *FlowNode) DeepCopy() *FlowNode {
	if in == nil {
		return nil
	}
	out := new(FlowNode)
	in.DeepCopyInto(out)
	return out
}
