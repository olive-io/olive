/*
Copyright 2023 The olive Authors

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

package backend

import (
	"os"
	"runtime/debug"
	"strings"

	"go.uber.org/zap"
)

const (
	ENV_VERIFY           = "OLIVE_VERIFY"
	ENV_VERIFY_ALL_VALUE = "all"
	ENV_VERIFY_LOCK      = "lock"
)

func ValidateCalledInsideApply(lg *zap.Logger) {
	if !verifyLockEnabled() {
		return
	}
	if !insideApply() {
		lg.Panic("Called outside of APPLY!", zap.Stack("stacktrace"))
	}
}

func ValidateCalledOutSideApply(lg *zap.Logger) {
	if !verifyLockEnabled() {
		return
	}
	if insideApply() {
		lg.Panic("Called inside of APPLY!", zap.Stack("stacktrace"))
	}
}

func ValidateCalledInsideUnittest(lg *zap.Logger) {
	if !verifyLockEnabled() {
		return
	}
	if !insideUnittest() {
		lg.Fatal("Lock called outside of unit test!", zap.Stack("stacktrace"))
	}
}

func verifyLockEnabled() bool {
	return os.Getenv(ENV_VERIFY) == ENV_VERIFY_ALL_VALUE || os.Getenv(ENV_VERIFY) == ENV_VERIFY_LOCK
}

func insideApply() bool {
	stackTraceStr := string(debug.Stack())
	return strings.Contains(stackTraceStr, ".applyEntries")
}

func insideUnittest() bool {
	stackTraceStr := string(debug.Stack())
	return strings.Contains(stackTraceStr, "_test.go") && !strings.Contains(stackTraceStr, "tests/")
}
