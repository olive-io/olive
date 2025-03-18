package docs

import (
	"embed"
	"io/fs"
)

//go:embed swagger
//go:embed openapi.yaml
var embedFs embed.FS

func GetSwagger() (fs.FS, error) {
	return fs.Sub(embedFs, "swagger")
}

func GetOpenYAML() ([]byte, error) {
	return fs.ReadFile(embedFs, "openapi.yaml")
}
