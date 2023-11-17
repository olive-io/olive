package version

import (
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"
)

var (
	// MinClusterVersion is the min cluster version this olive binary is compatible with.
	MinClusterVersion = "1.0.0"
	Version           = "1.0.0"
	APIVersion        = "unknown"

	// GitSHA Git SHA Value will be set during build
	GitSHA = "Not provided (use ./build instead of go build)"
)

func init() {
	ver, err := semver.NewVersion(Version)
	if err == nil {
		APIVersion = fmt.Sprintf("%d.%d", ver.Major, ver.Minor)
	}
}

type Versions struct {
	Server  string `json:"oliveserver"`
	Cluster string `json:"olivecluster"`
	// TODO: raft state machine version
}

// Cluster only keeps the major.minor.
func Cluster(v string) string {
	vs := strings.Split(v, ".")
	if len(vs) <= 2 {
		return v
	}
	return fmt.Sprintf("%s.%s", vs[0], vs[1])
}
