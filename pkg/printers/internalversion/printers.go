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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	krt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"

	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	api "github.com/olive-io/olive/apis/core"
	corev1 "github.com/olive-io/olive/apis/core/v1"
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
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(runnerColumnDefinitions, printRunner)
	_ = h.TableHandler(runnerColumnDefinitions, printRunnerList)

	regionColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the region"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(regionColumnDefinitions, printRegion)
	_ = h.TableHandler(regionColumnDefinitions, printRegionList)

	edgeColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the edge"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(edgeColumnDefinitions, printEdge)
	_ = h.TableHandler(edgeColumnDefinitions, printEdgeList)

	endpointColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the endpoint"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(endpointColumnDefinitions, printEndpoint)
	_ = h.TableHandler(endpointColumnDefinitions, printEndpointList)

	serviceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the service"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(serviceColumnDefinitions, printService)
	_ = h.TableHandler(serviceColumnDefinitions, printServiceList)

	namespaceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the namespace"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(namespaceColumnDefinitions, printNamespace)
	_ = h.TableHandler(namespaceColumnDefinitions, printNamespaceList)

	definitionColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the bpmn definition"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(definitionColumnDefinitions, printDefinition)
	_ = h.TableHandler(definitionColumnDefinitions, printDefinitionList)

	processColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the bpmn process"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(processColumnDefinitions, printProcess)
	_ = h.TableHandler(processColumnDefinitions, printProcessList)
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

func printRegion(obj *monv1.Region, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}

	row.Cells = append(row.Cells, obj.Name, obj.Status.Phase)

	return []metav1.TableRow{row}, nil
}

func printRegionList(list *monv1.RegionList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printRegion(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printEdge(obj *apidiscoveryv1.Edge, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}

	row.Cells = append(row.Cells, obj.Name)

	return []metav1.TableRow{row}, nil
}

func printEdgeList(list *apidiscoveryv1.EdgeList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printEdge(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printEndpoint(obj *apidiscoveryv1.Endpoint, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}

	row.Cells = append(row.Cells, obj.Name)

	return []metav1.TableRow{row}, nil
}

func printEndpointList(list *apidiscoveryv1.EndpointList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printEndpoint(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printService(obj *apidiscoveryv1.Service, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}

	row.Cells = append(row.Cells, obj.Name)

	return []metav1.TableRow{row}, nil
}

func printServiceList(list *apidiscoveryv1.ServiceList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printService(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printNamespace(obj *corev1.Namespace, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}
	row.Cells = append(row.Cells, obj.Name, string(obj.Status.Phase), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printNamespaceList(list *corev1.NamespaceList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printNamespace(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printDefinition(obj *corev1.Definition, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}
	row.Cells = append(row.Cells, obj.Name, string(obj.Status.Phase), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printDefinitionList(list *corev1.DefinitionList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printDefinition(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printProcess(obj *corev1.Process, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}
	row.Cells = append(row.Cells, obj.Name, string(obj.Status.Phase), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printProcessList(list *corev1.ProcessList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printProcess(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

// translateMicroTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateMicroTimestampSince(timestamp metav1.MicroTime) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

// translateTimestampUntil returns the elapsed time until timestamp in
// human-readable approximation.
func translateTimestampUntil(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Until(timestamp.Time))
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
