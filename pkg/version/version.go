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

package version

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	GitSHA    = "build ./build.sh"
	GitTag    = ""
	BuildDate = ""
)

func ReleaseVersion() string {
	var version string

	if GitTag != "" {
		version = GitTag
	}

	if GitSHA != "" {
		version += fmt.Sprintf("-%s", GitSHA)
	}

	if BuildDate != "" {
		version += fmt.Sprintf("-%s", BuildDate)
	}

	if version == "" {
		version = "latest"
	}

	return version
}

func GoV() string {
	v := strings.TrimPrefix(runtime.Version(), "go")
	if strings.Count(v, ".") > 1 {
		v = v[:strings.LastIndex(v, ".")]
	}
	return v
}

func GetVersionTemplate() string {
	var tpl string
	tpl += fmt.Sprintf("maco Version: %s\n", GitTag)
	tpl += fmt.Sprintf("Git SHA: %s\n", GitSHA)
	tpl += fmt.Sprintf("Go Version: %s\n", runtime.Version())
	tpl += fmt.Sprintf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	return tpl
}
