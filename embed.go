package moviedb

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var staticFiles embed.FS

// GetDistFS returns the embedded dist filesystem
func GetDistFS() (fs.FS, error) {
	return fs.Sub(staticFiles, "web/dist")
}
