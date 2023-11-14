//go:build linux

package netutil

import "testing"

func TestGetDefaultInterface(t *testing.T) {
	ifc, err := GetDefaultInterfaces()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("default network interfaces: %+v\n", ifc)
}

func TestGetDefaultHost(t *testing.T) {
	ip, err := GetDefaultHost()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("default ip: %v", ip)
}
