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

package internalversion

import (
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	krt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"

	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	api "github.com/olive-io/olive/apis/core"
	corev1 "github.com/olive-io/olive/apis/core/v1"
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
		{Name: "Hostname", Type: "string", Description: "The hostname of the olive-runner."},
		{Name: "PeerURL", Type: "string", Description: "The peer url of the olive-runner"},
		{Name: "ClientURL", Type: "string", Description: "The client url of the olive-runner"},
		{Name: "CPU", Type: "string", Description: "The CPU usage of the olive-runner."},
		{Name: "Memory", Type: "string", Description: "The memory usage of the olive-runner."},
		{Name: "Status", Type: "string", Description: "The status of the olive-runner"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(runnerColumnDefinitions, printRunner)
	_ = h.TableHandler(runnerColumnDefinitions, printRunnerList)

	regionColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Replica", Type: "string", Description: "The replicas of raft shard"},
		{Name: "Term", Type: "integer", Description: "The election term of raft shard"},
		{Name: "Status", Type: "string", Description: "The status of the region"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(regionColumnDefinitions, printRegion)
	_ = h.TableHandler(regionColumnDefinitions, printRegionList)

	edgeColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the edge"},
		{Name: "Created", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(edgeColumnDefinitions, printEdge)
	_ = h.TableHandler(edgeColumnDefinitions, printEdgeList)

	endpointColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the endpoint"},
		{Name: "Created", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(endpointColumnDefinitions, printEndpoint)
	_ = h.TableHandler(endpointColumnDefinitions, printEndpointList)

	serviceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the service"},
		{Name: "Created", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(serviceColumnDefinitions, printService)
	_ = h.TableHandler(serviceColumnDefinitions, printServiceList)

	namespaceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the namespace"},
		{Name: "Created", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(namespaceColumnDefinitions, printNamespace)
	_ = h.TableHandler(namespaceColumnDefinitions, printNamespaceList)

	definitionColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Version", Type: "integer", Description: "The latest version of definition"},
		{Name: "Region", Type: "string", Description: "The region of definition binding"},
		{Name: "Status", Type: "string", Description: "The status of the bpmn definition"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(definitionColumnDefinitions, printDefinition)
	_ = h.TableHandler(definitionColumnDefinitions, printDefinitionList)

	processColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Definition", Type: "string", Description: "The name of Definition"},
		{Name: "Version", Type: "integer", Description: "The version of the Definition"},
		{Name: "Process", Type: "string", Description: "The Bpmn Process process set"},
		{Name: "Region", Type: "string", Description: "The region of process binding"},
		{Name: "Status", Type: "string", Description: "The status of the bpmn process"},
		{Name: "Created", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(processColumnDefinitions, printProcess)
	_ = h.TableHandler(processColumnDefinitions, printProcessList)
}

func printRunner(obj *corev1.Runner, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}

	cpuUsage := fmt.Sprintf("%d", int64(obj.Status.CpuTotal))
	memoryUsage := humanize.Bytes(uint64(obj.Status.MemoryTotal))

	row.Cells = append(row.Cells, obj.Name,
		obj.Spec.Hostname,
		obj.Spec.PeerURL,
		obj.Spec.ClientURL,
		cpuUsage,
		memoryUsage,
		obj.Status.Phase,
		translateTimestampSince(obj.CreationTimestamp))

	return []metav1.TableRow{row}, nil
}

func printRunnerList(list *corev1.RunnerList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
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

func printRegion(obj *corev1.Region, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: krt.RawExtension{Object: obj},
	}

	replicas := []string{}
	for _, replica := range obj.Spec.InitialReplicas {
		id := fmt.Sprintf("%d", replica.Id)
		if obj.Spec.Leader == replica.Id {
			replicas = append([]string{id}, replicas...)
		} else {
			replicas = append(replicas, id)
		}
	}

	row.Cells = append(row.Cells,
		obj.Name,
		strings.Join(replicas, ","),
		obj.Status.Stat.Term,
		obj.Status.Phase,
		translateTimestampSince(obj.CreationTimestamp))

	return []metav1.TableRow{row}, nil
}

func printRegionList(list *corev1.RegionList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
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

	region := fmt.Sprintf("rn%d", obj.Spec.Region)

	row.Cells = append(row.Cells, obj.Name,
		obj.Spec.Version,
		region,
		string(obj.Status.Phase),
		translateTimestampSince(obj.CreationTimestamp))
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

	definition := obj.Spec.Definition
	version := obj.Spec.Version
	bpmnProcess := obj.Spec.BpmnProcess
	region := fmt.Sprintf("rn%d", obj.Status.Region)

	row.Cells = append(row.Cells,
		obj.Name,
		definition,
		version,
		bpmnProcess,
		region,
		string(obj.Status.Phase),
		translateTimestampSince(obj.CreationTimestamp))
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
