//go:build debug

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
)

func getTemplateFileByName(a string) string {
	b, err := os.ReadFile(path.Join("views", a))
	if err != nil {
		return fmt.Sprintf("Error loading template file: '%s'", err.Error())
	}
	return string(b)
}

func getAssetsFilesystem() fs.FS {
	return os.DirFS(path.Join("assets"))
}
