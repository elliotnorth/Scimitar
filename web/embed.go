package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var static embed.FS

func Files() fs.FS {
	files, err := fs.Sub(static, "static")
	if err != nil {
		panic(err)
	}
	return files
}
