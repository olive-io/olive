package flag

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/pflag"
)

var underscoreWarnings = make(map[string]bool)

// WordSepNormalizeFunc changes all flags that contain "_" separators
func WordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "_") {
		return pflag.NormalizedName(strings.Replace(name, "_", "-", -1))
	}
	return pflag.NormalizedName(name)
}

// WarnWordSepNormalizeFunc changes and warns for flags that contain "_" separators
func WarnWordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "_") {
		nname := strings.Replace(name, "_", "-", -1)
		if _, alreadyWarned := underscoreWarnings[name]; !alreadyWarned {
			slog.Warn(fmt.Sprintf("using an underscore in a flag name is not supported. %s has been converted to %s.", name, nname))
			underscoreWarnings[name] = true
		}

		return pflag.NormalizedName(nname)
	}
	return pflag.NormalizedName(name)
}
