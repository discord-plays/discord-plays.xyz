//go:build debug

package res

import (
	"fmt"
	"io/fs"
	"os"
	"path"
)

func GetTemplateFileByName(a string) string {
	b, err := os.ReadFile(path.Join("res/pages", a))
	if err != nil {
		return fmt.Sprintf("Error loading template file: '%s'", err.Error())
	}
	return string(b)
}

func GetAssetsFilesystem() fs.FS {
	return os.DirFS(path.Join("res/assets"))
}
