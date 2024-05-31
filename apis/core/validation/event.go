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

package validation

import (
	"fmt"
	"reflect"
	"time"

	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

const (
	ReportingInstanceLengthLimit = 128
	ActionLengthLimit            = 128
	ReasonLengthLimit            = 128
	NoteLengthLimit              = 1024
)

func ValidateEventCreate(event *corev1.Event, requestVersion schema.GroupVersion) field.ErrorList {
	// Make sure events always pass legacy validation.
	allErrs := legacyValidateEvent(event, requestVersion)
	if requestVersion == corev1.SchemeGroupVersion {
		// No further validation for backwards compatibility.
		return allErrs
	}

	// Strict validation applies to creation via events.k8s.io/v1 API and newer.
	allErrs = append(allErrs, ValidateObjectMeta(&event.ObjectMeta, true, apimachineryvalidation.NameIsDNSSubdomain, field.NewPath("metadata"))...)
	allErrs = append(allErrs, validateV1EventSeries(event)...)
	zeroTime := time.Time{}
	if event.EventTime.Time == zeroTime {
		allErrs = append(allErrs, field.Required(field.NewPath("eventTime"), ""))
	}
	if event.Type != corev1.EventTypeNormal && event.Type != corev1.EventTypeWarning {
		allErrs = append(allErrs, field.Invalid(field.NewPath("type"), "", fmt.Sprintf("has invalid value: %v", event.Type)))
	}
	if event.FirstTimestamp.Time != zeroTime {
		allErrs = append(allErrs, field.Invalid(field.NewPath("firstTimestamp"), "", "needs to be unset"))
	}
	if event.LastTimestamp.Time != zeroTime {
		allErrs = append(allErrs, field.Invalid(field.NewPath("lastTimestamp"), "", "needs to be unset"))
	}
	if event.Count != 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("count"), "", "needs to be unset"))
	}
	if event.Source.Component != "" || event.Source.Host != "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("source"), "", "needs to be unset"))
	}
	return allErrs
}

func ValidateEventUpdate(newEvent, oldEvent *corev1.Event, requestVersion schema.GroupVersion) field.ErrorList {
	// Make sure the new event always passes legacy validation.
	allErrs := legacyValidateEvent(newEvent, requestVersion)
	if requestVersion == corev1.SchemeGroupVersion {
		// No further validation for backwards compatibility.
		return allErrs
	}

	// Strict validation applies to update via events.k8s.io/v1 API and newer.
	allErrs = append(allErrs, ValidateObjectMetaUpdate(&newEvent.ObjectMeta, &oldEvent.ObjectMeta, field.NewPath("metadata"))...)
	// if the series was modified, validate the new data
	if !reflect.DeepEqual(newEvent.Series, oldEvent.Series) {
		allErrs = append(allErrs, validateV1EventSeries(newEvent)...)
	}

	allErrs = append(allErrs, ValidateImmutableField(newEvent.InvolvedObject, oldEvent.InvolvedObject, field.NewPath("involvedObject"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.Reason, oldEvent.Reason, field.NewPath("reason"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.Message, oldEvent.Message, field.NewPath("message"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.Source, oldEvent.Source, field.NewPath("source"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.FirstTimestamp, oldEvent.FirstTimestamp, field.NewPath("firstTimestamp"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.LastTimestamp, oldEvent.LastTimestamp, field.NewPath("lastTimestamp"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.Count, oldEvent.Count, field.NewPath("count"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.Reason, oldEvent.Reason, field.NewPath("reason"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.Type, oldEvent.Type, field.NewPath("type"))...)

	// Disallow changes to eventTime greater than microsecond-level precision.
	// Tolerating sub-microsecond changes is required to tolerate updates
	// from clients that correctly truncate to microsecond-precision when serializing,
	// or from clients built with incorrect nanosecond-precision protobuf serialization.
	// See https://github.com/kubernetes/kubernetes/issues/111928
	newTruncated := newEvent.EventTime.Truncate(time.Microsecond).UTC()
	oldTruncated := oldEvent.EventTime.Truncate(time.Microsecond).UTC()
	if newTruncated != oldTruncated {
		allErrs = append(allErrs, ValidateImmutableField(newEvent.EventTime, oldEvent.EventTime, field.NewPath("eventTime"))...)
	}

	allErrs = append(allErrs, ValidateImmutableField(newEvent.Action, oldEvent.Action, field.NewPath("action"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.Related, oldEvent.Related, field.NewPath("related"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.ReportingController, oldEvent.ReportingController, field.NewPath("reportingController"))...)
	allErrs = append(allErrs, ValidateImmutableField(newEvent.ReportingInstance, oldEvent.ReportingInstance, field.NewPath("reportingInstance"))...)

	return allErrs
}

func validateV1EventSeries(event *corev1.Event) field.ErrorList {
	allErrs := field.ErrorList{}
	zeroTime := time.Time{}
	if event.Series != nil {
		if event.Series.Count < 2 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("series.count"), "", "should be at least 2"))
		}
		if event.Series.LastObservedTime.Time == zeroTime {
			allErrs = append(allErrs, field.Required(field.NewPath("series.lastObservedTime"), ""))
		}
	}
	return allErrs
}

// legacyValidateEvent makes sure that the event makes sense.
func legacyValidateEvent(event *corev1.Event, requestVersion schema.GroupVersion) field.ErrorList {
	allErrs := field.ErrorList{}
	// Because go
	zeroTime := time.Time{}

	reportingControllerFieldName := "reportingController"
	if requestVersion == corev1.SchemeGroupVersion {
		reportingControllerFieldName = "reportingComponent"
	}

	// "New" Events need to have EventTime set, so it's validating old object.
	if event.EventTime.Time == zeroTime {
		// Make sure event.Namespace and the involvedInvolvedObject.Namespace agree
		if len(event.InvolvedObject.Namespace) == 0 {
			// event.Namespace must also be empty (or "default", for compatibility with old clients)
			if event.Namespace != metav1.NamespaceNone && event.Namespace != metav1.NamespaceDefault {
				allErrs = append(allErrs, field.Invalid(field.NewPath("involvedObject", "namespace"), event.InvolvedObject.Namespace, "does not match event.namespace"))
			}
		} else {
			// event namespace must match
			if event.Namespace != event.InvolvedObject.Namespace {
				allErrs = append(allErrs, field.Invalid(field.NewPath("involvedObject", "namespace"), event.InvolvedObject.Namespace, "does not match event.namespace"))
			}
		}

	} else {
		if len(event.InvolvedObject.Namespace) == 0 && event.Namespace != metav1.NamespaceDefault && event.Namespace != metav1.NamespaceSystem {
			allErrs = append(allErrs, field.Invalid(field.NewPath("involvedObject", "namespace"), event.InvolvedObject.Namespace, "does not match event.namespace"))
		}
		if len(event.ReportingController) == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath(reportingControllerFieldName), ""))
		}
		allErrs = append(allErrs, ValidateQualifiedName(event.ReportingController, field.NewPath(reportingControllerFieldName))...)
		if len(event.ReportingInstance) == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("reportingInstance"), ""))
		}
		if len(event.ReportingInstance) > ReportingInstanceLengthLimit {
			allErrs = append(allErrs, field.Invalid(field.NewPath("reportingInstance"), "", fmt.Sprintf("can have at most %v characters", ReportingInstanceLengthLimit)))
		}
		if len(event.Action) == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("action"), ""))
		}
		if len(event.Action) > ActionLengthLimit {
			allErrs = append(allErrs, field.Invalid(field.NewPath("action"), "", fmt.Sprintf("can have at most %v characters", ActionLengthLimit)))
		}
		if len(event.Reason) == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("reason"), ""))
		}
		if len(event.Reason) > ReasonLengthLimit {
			allErrs = append(allErrs, field.Invalid(field.NewPath("reason"), "", fmt.Sprintf("can have at most %v characters", ReasonLengthLimit)))
		}
		if len(event.Message) > NoteLengthLimit {
			allErrs = append(allErrs, field.Invalid(field.NewPath("message"), "", fmt.Sprintf("can have at most %v characters", NoteLengthLimit)))
		}
	}

	for _, msg := range validation.IsDNS1123Subdomain(event.Namespace) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("namespace"), event.Namespace, msg))
	}
	return allErrs
}
