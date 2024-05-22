package main // nolint:revive

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
)

var bin string

var cmd = cobra.Command{
	Use:     "olive-runtime-gen",
	Short:   "run code generators",
	PreRunE: preRunE,
	RunE:    runE,
}

func preRunE(cmd *cobra.Command, args []string) error {
	if module == "" {
		return fmt.Errorf("must specify module")
	}
	if len(versions) == 0 {
		return fmt.Errorf("must specify versions")
	}
	return nil
}

func runE(cmd *cobra.Command, args []string) error {
	var err error

	// get the location the generators are installed
	bin = os.Getenv("GOBIN")
	if bin == "" {
		bin = filepath.Join(os.Getenv("GOPATH"), "bin")
	}
	// install the generators
	if install {
		for _, gen := range generators {
			// nolint:gosec
			err := run(exec.Command("go", "install", path.Join("k8s.io/code-generator/cmd", gen)))
			if err != nil {
				return err
			}
		}
	}

	for _, input := range versions {
		if err = doGen(input); err != nil {
			return err
		}
	}
	return nil
}

func doGen(input string) error {
	gen := map[string]bool{}
	for _, g := range generators {
		gen[g] = true
	}

	if gen["deepcopy-gen"] {
		err := run(getCmd("deepcopy-gen",
			"--output-file", "zz_generated.deepcopy.go",
			"--bounding-dirs", path.Join(module, "apis"), input))
		if err != nil {
			return err
		}
	}

	if gen["openapi-gen"] {
		err := run(getCmd("openapi-gen",
			"--output-file", "zz_generated.openapi.go",
			"--output-dir", path.Join("client/generated/openapi"),
			"--output-pkg", "module",
			input))
		if err != nil {
			return err
		}
	}

	//gopath := filepath.Join(os.Getenv("GOPATH"), "src")
	//if gen["go-to-protobuf"] {
	//	err := run(getCmd("go-to-protobuf",
	//		fmt.Sprintf("--proto-import=%s/kubernetes/kubernetes/third_party/protobuf", gopath),
	//		"--packages", inputs))
	//	if err != nil {
	//		return err
	//	}
	//}

	if gen["client-gen"] {
		inputBase := ""
		versionsInputs := input
		// e.g. base = "example.io/foo/api", strippedVersions = "v1,v1beta1"
		// e.g. base = "example.io/foo/pkg/apis", strippedVersions = "test/v1,test/v1beta1"
		if base, strippedVersions, ok := findInputBase(module, versions); ok {
			inputBase = base
			versionsInputs = strings.Join(strippedVersions, ",")
		}
		err := run(getCmd("client-gen",
			"--clientset-name", "versioned", "--input-base", inputBase,
			"--input", versionsInputs,
			"--output-pkg", path.Join("client/generated/clientset"),
			"--output-dir", module))
		if err != nil {
			return err
		}
	}

	if gen["lister-gen"] {
		err := run(getCmd("lister-gen",
			"--output-pkg", module,
			"--output-dir", path.Join("client/generated/listers"),
			input))
		if err != nil {
			return err
		}
	}

	if gen["informer-gen"] {
		err := run(getCmd("informer-gen",
			"--versioned-clientset-package", path.Join("client/generated/clientset/versioned"),
			"--listers-package", path.Join(module, "client/generated/listers"),
			"--output-pkg", module,
			"--output-dir", path.Join("client/generated/informers"),
			input))
		if err != nil {
			return err
		}
	}

	return nil
}

var (
	generators     []string
	header         string
	module         string
	versions       []string
	clean, install bool
)

func main() {
	cmd.Flags().BoolVar(&clean, "clean", true, "Delete temporary directory for code generation.")

	options := []string{"client-gen", "deepcopy-gen", "informer-gen", "lister-gen", "openapi-gen", "go-to-protobuf"}
	defaultGen := []string{"deepcopy-gen", "openapi-gen"}
	cmd.Flags().StringSliceVarP(&generators, "generator", "g",
		defaultGen, fmt.Sprintf("Code generator to install and run.  Options: %v.", options))
	defaultBoilerplate := filepath.Join("hack", "boilerplate.go.txt")
	cmd.Flags().StringVar(&header, "go-header-file", defaultBoilerplate,
		"File containing boilerplate header text. The string YEAR will be replaced with the current 4-digit year.")
	cmd.Flags().BoolVar(&install, "install-generators", false, "Go get the generators")

	var defaultModule string
	cwd, _ := os.Getwd()
	if modRoot := findModuleRoot(cwd); modRoot != "" {
		if b, err := os.ReadFile(filepath.Clean(path.Join(modRoot, "go.mod"))); err == nil {
			defaultModule = modfile.ModulePath(b)
		}
	}
	cmd.Flags().StringVar(&module, "module", defaultModule, "Go module of the apiserver.")

	// calculate the versions
	var defaultVersions []string
	if files, err := os.ReadDir(filepath.Join("apis")); err == nil {
		for _, f := range files {
			if f.IsDir() {
				versionFiles, err := os.ReadDir(filepath.Join("apis", f.Name()))
				if err != nil {
					log.Fatal(err)
				}
				for _, v := range versionFiles {
					if v.IsDir() {
						match := versionRegexp.MatchString(v.Name())
						if !match {
							continue
						}
						defaultVersions = append(defaultVersions, path.Join(module, "apis", f.Name(), v.Name()))
					}
				}
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "cannot parse api versions: %v\n", err)
	}
	cmd.Flags().StringSliceVar(
		&versions, "versions", defaultVersions, "Go packages of API versions to generate code for.")

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var versionRegexp = regexp.MustCompile("^v[0-9]+((alpha|beta)?[0-9]+)?$")

func run(cmd *exec.Cmd) error {
	cmd.Env = os.Environ()
	fmt.Println(strings.Join(cmd.Args, " "))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func getCmd(cmd string, args ...string) *exec.Cmd {
	// nolint:gosec
	e := exec.Command(filepath.Join(bin, cmd), "--go-header-file", header)

	e.Args = append(e.Args, args...)
	return e
}

func findModuleRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parentDIR := path.Dir(dir)
		if parentDIR == dir {
			break
		}
		dir = parentDIR
	}
	return ""
}

func findInputBase(module string, versions []string) (string, []string, bool) {
	//if allHasPrefix(filepath.Join(module, "api"), versions) {
	//	base := filepath.Join(module, "api")
	//	return base, allTrimPrefix(base+"/", versions), true
	//}
	if allHasPrefix(filepath.Join(module, "apis"), versions) {
		base := filepath.Join(module, "apis")
		return base, allTrimPrefix(base+"/", versions), true
	}
	return "", nil, false
}

func allHasPrefix(prefix string, paths []string) bool {
	for _, p := range paths {
		if !strings.HasPrefix(p, prefix) {
			return false
		}
	}
	return true
}

func allTrimPrefix(prefix string, versions []string) []string {
	vs := make([]string, 0)
	for _, v := range versions {
		vs = append(vs, strings.TrimPrefix(v, prefix))
	}
	return vs
}
