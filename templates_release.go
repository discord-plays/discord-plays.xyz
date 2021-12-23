//go:build !debug

package main

import (
	"embed"
	"fmt"
	"path"
)

var (
	//go:embed views
	viewsFiles embed.FS
)

func getTemplateFileByName(a string) string {
	b, err := viewsFiles.ReadFile(path.Join("views", a))
	if err != nil {
		return fmt.Sprintf("Error loading template file: '%s'", err.Error())
	}
	return string(b)
}
