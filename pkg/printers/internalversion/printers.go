/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package internalversion

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	krt "k8s.io/apimachinery/pkg/runtime"

	api "github.com/olive-io/olive/apis/core"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	"github.com/olive-io/olive/pkg/printers"
)

const (
	// labelNodeRolePrefix is a label prefix for runner roles
	labelRunnerRolePrefix = "runner-role.olive.io/"

	// runnerLabelRole specifies the role of a runner
	runnerLabelRole = "olive.io/role"
)

// AddHandlers adds print handlers for default Kubernetes types dealing with internal versions.
func AddHandlers(h printers.PrintHandler) {
	runnerColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the node"},
	}

	_ = h.TableHandler(runnerColumnDefinitions, printRunner)
	_ = h.TableHandler(runnerColumnDefinitions, printRunnerList)
}

func printRunner(obj *monv1.Runner, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}

	ips := []string{}
	for _, url := range obj.Spec.ClientURLs {
		ips = append(ips, url)
	}

	row.Cells = append(row.Cells, obj.Name, strings.Join(ips, ","), obj.Status.Phase)

	return []metav1.TableRow{row}, nil
}

func printRunnerList(list *monv1.RunnerList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printRunner(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printBoolPtr(value *bool) string {
	if value != nil {
		return printBool(*value)
	}

	return "<unset>"
}

func printBool(value bool) string {
	if value {
		return "True"
	}

	return "False"
}

// SortableResourceNames - An array of sortable resource names
type SortableResourceNames []api.ResourceName

func (list SortableResourceNames) Len() int {
	return len(list)
}

func (list SortableResourceNames) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list SortableResourceNames) Less(i, j int) bool {
	return list[i] < list[j]
}
