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
