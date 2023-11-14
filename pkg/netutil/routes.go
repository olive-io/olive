//go:build !linux

package netutil

import (
	"fmt"
	"runtime"
)

// GetDefaultHost fetches the a resolvable name that corresponds
// to the machine's default routable interface
func GetDefaultHost() (string, error) {
	return "", fmt.Errorf("default host not supported on %s_%s", runtime.GOOS, runtime.GOARCH)
}

// GetDefaultInterfaces fetches the device name of default routable interface.
func GetDefaultInterfaces() (map[string]uint8, error) {
	return nil, fmt.Errorf("default host not supported on %s_%s", runtime.GOOS, runtime.GOARCH)
}
