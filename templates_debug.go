//go:build debug

package main

import (
	"fmt"
	"os"
	"path"
)

func getTemplateFileByName(a string) string {
	b, err := os.ReadFile(path.Join(os.Getenv("DEV_FOLDER"), "views", a))
	if err != nil {
		return fmt.Sprintf("Error loading template file: '%s'", err.Error())
	}
	return string(b)
}
