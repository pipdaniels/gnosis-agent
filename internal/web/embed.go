package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFS embed.FS

//go:embed templates
var templatesFS embed.FS

// GetStaticFS returns the embedded static filesystem
func GetStaticFS() http.FileSystem {
	// The embed directive makes "static" the root of the file system within staticFS.
	// We want to serve files relative to that "static" directory.
	// However, `embed` creates a file system where the root contains the matched pattern.
	// So staticFS contains a single entry "static" at the root.
	// To serve the contents of "static", we sub into it.
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("failed to create static sub-filesystem: " + err.Error())
	}
	return http.FS(staticSub)
}

// GetTemplatesFS returns the embedded templates filesystem
func GetTemplatesFS() fs.FS {
	templatesSub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic("failed to create templates sub-filesystem: " + err.Error())
	}
	return templatesSub
}
