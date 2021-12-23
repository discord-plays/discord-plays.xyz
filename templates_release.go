//go:build !debug

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
)

var (
	//go:embed views
	viewsFiles embed.FS
	//go:embed assets
	assetsFiles embed.FS
)

func getTemplateFileByName(a string) string {
	b, err := viewsFiles.ReadFile(path.Join("views", a))
	if err != nil {
		return fmt.Sprintf("Error loading template file: '%s'", err.Error())
	}
	return string(b)
}

func getAssetsFilesystem() fs.FS {
	f, err := fs.Sub(assetsFiles, "assets")
	if err != nil {
		return nil
	}
	return f
}
