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

package framework

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/sets"
)

var errorStatus = NewStatus(Error, "internal error")
var statusWithErr = AsStatus(errors.New("internal error"))

func TestStatus(t *testing.T) {
	tests := []struct {
		name              string
		status            *Status
		expectedCode      Code
		expectedMessage   string
		expectedIsSuccess bool
		expectedIsWait    bool
		expectedIsSkip    bool
		expectedAsError   error
	}{
		{
			name:              "success status",
			status:            NewStatus(Success, ""),
			expectedCode:      Success,
			expectedMessage:   "",
			expectedIsSuccess: true,
			expectedIsWait:    false,
			expectedIsSkip:    false,
			expectedAsError:   nil,
		},
		{
			name:              "wait status",
			status:            NewStatus(Wait, ""),
			expectedCode:      Wait,
			expectedMessage:   "",
			expectedIsSuccess: false,
			expectedIsWait:    true,
			expectedIsSkip:    false,
			expectedAsError:   nil,
		},
		{
			name:              "error status",
			status:            NewStatus(Error, "unknown error"),
			expectedCode:      Error,
			expectedMessage:   "unknown error",
			expectedIsSuccess: false,
			expectedIsWait:    false,
			expectedIsSkip:    false,
			expectedAsError:   errors.New("unknown error"),
		},
		{
			name:              "skip status",
			status:            NewStatus(Skip, ""),
			expectedCode:      Skip,
			expectedMessage:   "",
			expectedIsSuccess: false,
			expectedIsWait:    false,
			expectedIsSkip:    true,
			expectedAsError:   nil,
		},
		{
			name:              "nil status",
			status:            nil,
			expectedCode:      Success,
			expectedMessage:   "",
			expectedIsSuccess: true,
			expectedIsSkip:    false,
			expectedAsError:   nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.status.Code() != test.expectedCode {
				t.Errorf("expect status.Code() returns %v, but %v", test.expectedCode, test.status.Code())
			}

			if test.status.Message() != test.expectedMessage {
				t.Errorf("expect status.Message() returns %v, but %v", test.expectedMessage, test.status.Message())
			}

			if test.status.IsSuccess() != test.expectedIsSuccess {
				t.Errorf("expect status.IsSuccess() returns %v, but %v", test.expectedIsSuccess, test.status.IsSuccess())
			}

			if test.status.IsWait() != test.expectedIsWait {
				t.Errorf("status.IsWait() returns %v, but want %v", test.status.IsWait(), test.expectedIsWait)
			}

			if test.status.IsSkip() != test.expectedIsSkip {
				t.Errorf("status.IsSkip() returns %v, but want %v", test.status.IsSkip(), test.expectedIsSkip)
			}

			if test.status.AsError() == test.expectedAsError {
				return
			}

			if test.status.AsError().Error() != test.expectedAsError.Error() {
				t.Errorf("expect status.AsError() returns %v, but %v", test.expectedAsError, test.status.AsError())
			}
		})
	}
}

func TestPreFilterResultMerge(t *testing.T) {
	tests := map[string]struct {
		receiver *PreFilterResult
		in       *PreFilterResult
		want     *PreFilterResult
	}{
		"all nil": {},
		"nil receiver empty input": {
			in:   &PreFilterResult{RunnerNames: sets.New[string]()},
			want: &PreFilterResult{RunnerNames: sets.New[string]()},
		},
		"empty receiver nil input": {
			receiver: &PreFilterResult{RunnerNames: sets.New[string]()},
			want:     &PreFilterResult{RunnerNames: sets.New[string]()},
		},
		"empty receiver empty input": {
			receiver: &PreFilterResult{RunnerNames: sets.New[string]()},
			in:       &PreFilterResult{RunnerNames: sets.New[string]()},
			want:     &PreFilterResult{RunnerNames: sets.New[string]()},
		},
		"nil receiver populated input": {
			in:   &PreFilterResult{RunnerNames: sets.New("node1")},
			want: &PreFilterResult{RunnerNames: sets.New("node1")},
		},
		"empty receiver populated input": {
			receiver: &PreFilterResult{RunnerNames: sets.New[string]()},
			in:       &PreFilterResult{RunnerNames: sets.New("node1")},
			want:     &PreFilterResult{RunnerNames: sets.New[string]()},
		},

		"populated receiver nil input": {
			receiver: &PreFilterResult{RunnerNames: sets.New("node1")},
			want:     &PreFilterResult{RunnerNames: sets.New("node1")},
		},
		"populated receiver empty input": {
			receiver: &PreFilterResult{RunnerNames: sets.New("node1")},
			in:       &PreFilterResult{RunnerNames: sets.New[string]()},
			want:     &PreFilterResult{RunnerNames: sets.New[string]()},
		},
		"populated receiver and input": {
			receiver: &PreFilterResult{RunnerNames: sets.New("node1", "node2")},
			in:       &PreFilterResult{RunnerNames: sets.New("node2", "node3")},
			want:     &PreFilterResult{RunnerNames: sets.New("node2")},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.receiver.Merge(test.in)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("unexpected diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestIsStatusEqual(t *testing.T) {
	tests := []struct {
		name string
		x, y *Status
		want bool
	}{
		{
			name: "two nil should be equal",
			x:    nil,
			y:    nil,
			want: true,
		},
		{
			name: "nil should be equal to success status",
			x:    nil,
			y:    NewStatus(Success),
			want: true,
		},
		{
			name: "nil should not be equal with status except success",
			x:    nil,
			y:    NewStatus(Error, "internal error"),
			want: false,
		},
		{
			name: "one status should be equal to itself",
			x:    errorStatus,
			y:    errorStatus,
			want: true,
		},
		{
			name: "same type statuses without reasons should be equal",
			x:    NewStatus(Success),
			y:    NewStatus(Success),
			want: true,
		},
		{
			name: "statuses with same message should be equal",
			x:    NewStatus(Unschedulable, "unschedulable"),
			y:    NewStatus(Unschedulable, "unschedulable"),
			want: true,
		},
		{
			name: "error statuses with same message should be equal",
			x:    NewStatus(Error, "error"),
			y:    NewStatus(Error, "error"),
			want: true,
		},
		{
			name: "statuses with different reasons should not be equal",
			x:    NewStatus(Unschedulable, "unschedulable"),
			y:    NewStatus(Unschedulable, "unschedulable", "injected filter status"),
			want: false,
		},
		{
			name: "statuses with different codes should not be equal",
			x:    NewStatus(Error, "internal error"),
			y:    NewStatus(Unschedulable, "internal error"),
			want: false,
		},
		{
			name: "wrap error status should be equal with original one",
			x:    statusWithErr,
			y:    AsStatus(fmt.Errorf("error: %w", statusWithErr.AsError())),
			want: true,
		},
		{
			name: "statues with different errors that have the same message shouldn't be equal",
			x:    AsStatus(errors.New("error")),
			y:    AsStatus(errors.New("error")),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.x.Equal(tt.y); got != tt.want {
				t.Errorf("cmp.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
