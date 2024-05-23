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

package printers

import (
	"fmt"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type TestPrintType struct {
	Data string
}

func (obj *TestPrintType) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (obj *TestPrintType) DeepCopyObject() runtime.Object {
	if obj == nil {
		return nil
	}
	clone := *obj
	return &clone
}

func PrintCustomType(obj *TestPrintType, options GenerateOptions) ([]metav1beta1.TableRow, error) {
	return []metav1beta1.TableRow{{Cells: []interface{}{obj.Data}}}, nil
}

func ErrorPrintHandler(obj *TestPrintType, options GenerateOptions) ([]metav1beta1.TableRow, error) {
	return nil, fmt.Errorf("ErrorPrintHandler error")
}

func TestCustomTypePrinting(t *testing.T) {
	columns := []metav1beta1.TableColumnDefinition{{Name: "Data"}}
	generator := NewTableGenerator()
	err := generator.TableHandler(columns, PrintCustomType)
	if err != nil {
		t.Fatalf("An error occurred when adds a print handler with a given set of columns: %#v", err)
	}

	obj := TestPrintType{"test object"}
	table, err := generator.GenerateTable(&obj, GenerateOptions{})
	if err != nil {
		t.Fatalf("An error occurred generating the table for custom type: %#v", err)
	}

	expectedTable := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{{Name: "Data"}},
		Rows:              []metav1.TableRow{{Cells: []interface{}{"test object"}}},
	}
	if !reflect.DeepEqual(expectedTable, table) {
		t.Errorf("Error generating table from custom type. Expected (%#v), got (%#v)", expectedTable, table)
	}
}

func TestPrintHandlerError(t *testing.T) {
	columns := []metav1beta1.TableColumnDefinition{{Name: "Data"}}
	generator := NewTableGenerator()
	err := generator.TableHandler(columns, ErrorPrintHandler)
	if err != nil {
		t.Fatalf("An error occurred when adds a print handler with a given set of columns: %#v", err)
	}

	obj := TestPrintType{"test object"}
	_, err = generator.GenerateTable(&obj, GenerateOptions{})
	if err == nil || err.Error() != "ErrorPrintHandler error" {
		t.Errorf("Did not get the expected error: %#v", err)
	}
}
