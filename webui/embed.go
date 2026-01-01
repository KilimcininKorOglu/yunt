// Package webui provides the embedded web UI files and a handler to serve them.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
)

//go:embed dist/*
var distFS embed.FS

// Handler returns an Echo handler function that serves the embedded web UI.
// It handles SPA routing by serving index.html for any path that doesn't match
// a static asset.
func Handler() echo.HandlerFunc {
	// Create a sub-filesystem rooted at "dist"
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("failed to create sub filesystem: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(subFS))

	return func(c echo.Context) error {
		reqPath := c.Request().URL.Path

		// Clean the path
		reqPath = path.Clean("/" + reqPath)
		if reqPath == "/" {
			reqPath = "/index.html"
		}

		// Remove leading slash for fs operations
		fsPath := strings.TrimPrefix(reqPath, "/")

		// Check if the file exists in the embedded filesystem
		if _, err := fs.Stat(subFS, fsPath); err == nil {
			// File exists, serve it with appropriate cache headers
			setCacheHeaders(c, fsPath)
			fileServer.ServeHTTP(c.Response(), c.Request())
			return nil
		}

		// File doesn't exist, serve index.html for SPA routing
		c.Request().URL.Path = "/index.html"
		fileServer.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// setCacheHeaders sets appropriate cache headers based on file type.
func setCacheHeaders(c echo.Context, filePath string) {
	// Immutable assets (with hash in filename) can be cached forever
	// These typically include: .js, .css files with content hash
	if isImmutableAsset(filePath) {
		// Cache for 1 year (immutable)
		c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}

	// HTML files should not be cached aggressively
	if strings.HasSuffix(filePath, ".html") {
		c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		return
	}

	// Other static assets (fonts, images) - cache for 1 week
	c.Response().Header().Set("Cache-Control", "public, max-age=604800")
}

// isImmutableAsset checks if the file is a hashed/immutable asset.
// SvelteKit generates files like: _app/immutable/assets/xxx-hash.css
func isImmutableAsset(filePath string) bool {
	// SvelteKit puts immutable assets in _app/immutable/
	if strings.Contains(filePath, "_app/immutable/") {
		return true
	}

	// Also check for common patterns with content hashes
	// e.g., main.abc123.js, styles.def456.css
	ext := path.Ext(filePath)
	if ext == ".js" || ext == ".css" {
		base := strings.TrimSuffix(path.Base(filePath), ext)
		// If the base name contains a dot, it might have a hash
		if strings.Contains(base, ".") || strings.Contains(base, "-") {
			return true
		}
	}

	return false
}

// IsAvailable returns true if the embedded web UI has content.
// This is useful for checking if the UI was built before embedding.
// It returns true only if index.html exists, ignoring placeholder files like .gitkeep.
func IsAvailable() bool {
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		return false
	}

	// Check if index.html exists - this is the main entry point for the SPA
	_, err = fs.Stat(subFS, "index.html")
	return err == nil
}
