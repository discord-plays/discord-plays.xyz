//go:build !debug

package res

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
)

var (
	//go:embed pages
	viewsFiles embed.FS
	//go:embed assets
	assetsFiles embed.FS
)

func GetTemplateFileByName(a string) string {
	b, err := viewsFiles.ReadFile(path.Join("pages", a))
	if err != nil {
		return fmt.Sprintf("Error loading template file: '%s'", err.Error())
	}
	return string(b)
}

func GetAssetsFilesystem() fs.FS {
	f, err := fs.Sub(assetsFiles, "assets")
	if err != nil {
		return nil
	}
	return f
}
